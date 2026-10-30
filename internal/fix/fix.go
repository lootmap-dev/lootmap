// Package fix implements the Phase 6 remediation engine: planning, live-session
// protection, 0600 backups with 7-day expiry, atomic writes, and byte-exact undo.
package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
)

// LiveSessionWindow is 60 seconds (SPEC.md: AI transcript files modified in the
// last 60 seconds are not touched unless --force is passed).
const LiveSessionWindow = 60 * time.Second

// FileChange describes a planned modification to a single file.
type FileChange struct {
	FixerID     string `json:"fixerId"`
	RelPath     string `json:"relPath"`     // relative to home
	ShortPath   string `json:"shortPath"`   // ~/...
	Description string `json:"description"` // line-by-line preview summary
	NewContent  []byte `json:"-"`           // nil when only permissions change
	Chmod600    bool   `json:"chmod600"`
}

// Plan holds all FileChanges proposed by one or more Fixers.
type Plan struct {
	Changes []FileChange `json:"changes"`
	Skipped []string     `json:"skipped,omitempty"`
}

// Fixer plans and applies one category of safe automatic remediation.
type Fixer interface {
	ID() string
	Plan(e env.Env, findings []probes.Finding, opts Options) (Plan, error)
}

// Options controls fixer planning and application.
type Options struct {
	Force            bool
	Now              func() time.Time
	FailBeforeRename bool // test hook for simulated interrupted writes
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// AllFixers returns all v0.1 Fixers in deterministic execution order.
func AllFixers() []Fixer {
	return []Fixer{
		RedactFixer{id: "redact-transcripts", targetProbe: "ai-transcripts"},
		RedactFixer{id: "redact-shell-history", targetProbe: "shell-history"},
		GitCredentialHelperFixer{},
		NPMIgnoreScriptsFixer{},
		SSHDisableForwardAgentFixer{},
		PermissionsFixer{id: "ssh-permissions"},
		PermissionsFixer{id: "file-permissions-600"},
	}
}

// BuildPlan runs all v0.1 Fixers against findings and merges their FileChanges.
func BuildPlan(e env.Env, findings []probes.Finding, opts Options) (Plan, error) {
	var combined Plan
	for _, fx := range AllFixers() {
		p, err := fx.Plan(e, findings, opts)
		if err != nil {
			return Plan{}, err
		}
		combined.Changes = append(combined.Changes, p.Changes...)
		combined.Skipped = append(combined.Skipped, p.Skipped...)
	}
	return coalescePlan(combined), nil
}

// coalescePlan merges multiple changes to the same file so atomic write +
// backup operate cleanly on each target file.
func coalescePlan(p Plan) Plan {
	byPath := map[string]int{}
	var out []FileChange
	for _, ch := range p.Changes {
		if idx, ok := byPath[ch.RelPath]; ok {
			existing := &out[idx]
			if ch.NewContent != nil {
				existing.NewContent = ch.NewContent
			}
			if ch.Chmod600 {
				existing.Chmod600 = true
			}
			if !strings.Contains(existing.Description, ch.Description) {
				existing.Description += "; " + ch.Description
			}
			continue
		}
		byPath[ch.RelPath] = len(out)
		out = append(out, ch)
	}
	return Plan{Changes: out, Skipped: p.Skipped}
}

// Apply creates a backup snapshot in ~/.lootmap/backups/<id>/ and applies every
// FileChange in plan atomically.
func Apply(e env.Env, plan Plan, bmgr *BackupManager, opts Options) (BackupManifest, error) {
	if len(plan.Changes) == 0 {
		return BackupManifest{}, nil
	}
	rels := make([]string, 0, len(plan.Changes))
	for _, ch := range plan.Changes {
		rels = append(rels, ch.RelPath)
	}
	manifest, err := bmgr.CreateSnapshot(rels)
	if err != nil {
		return BackupManifest{}, err
	}

	for _, ch := range plan.Changes {
		absPath := filepath.Join(e.Home(), filepath.FromSlash(ch.RelPath))
		mode := os.FileMode(0o600)
		if !ch.Chmod600 {
			if st, err := os.Stat(absPath); err == nil {
				mode = st.Mode().Perm()
			}
		}
		if ch.NewContent != nil {
			if err := AtomicWrite(absPath, ch.NewContent, mode, opts.FailBeforeRename); err != nil {
				return manifest, err
			}
		} else if ch.Chmod600 {
			if err := tightenPermissionsOS(absPath); err != nil {
				return manifest, err
			}
		}
	}
	return manifest, nil
}

// RedactFixer replaces matched secrets in AI transcripts or shell history with
// [REDACTED:<kind>].
type RedactFixer struct {
	id          string
	targetProbe string
}

func (r RedactFixer) ID() string { return r.id }

func (r RedactFixer) Plan(e env.Env, findings []probes.Finding, opts Options) (Plan, error) {
	files := map[string]bool{}
	for _, f := range findings {
		if f.Location.Probe == r.targetProbe {
			rel := normalizeRel(f.Location.Path)
			if rel != "" {
				files[rel] = true
			}
		}
	}
	if len(files) == 0 {
		return Plan{}, nil
	}

	rawRegs, err := compileRedactionRegexes()
	if err != nil {
		return Plan{}, err
	}

	var plan Plan
	for rel := range files {
		absPath := filepath.Join(e.Home(), filepath.FromSlash(rel))
		st, err := os.Stat(absPath)
		if err != nil {
			continue
		}
		// Live session protection for AI transcripts modified within 60s.
		if r.targetProbe == "ai-transcripts" && !opts.Force {
			if opts.now().Sub(st.ModTime()) < LiveSessionWindow {
				plan.Skipped = append(plan.Skipped, fmt.Sprintf("~/%s (modified <60s ago; use --force to override)", rel))
				continue
			}
		}

		orig, err := os.ReadFile(filepath.Clean(absPath)) // #nosec G304 -- validated relative path inside Home
		if err != nil {
			continue
		}
		redacted, count := redactBytes(orig, rawRegs)
		if count == 0 {
			continue
		}
		plan.Changes = append(plan.Changes, FileChange{
			FixerID:     r.id,
			RelPath:     rel,
			ShortPath:   "~/" + rel,
			Description: fmt.Sprintf("redact %d secret(s) -> [REDACTED:<kind>]", count),
			NewContent:  redacted,
			Chmod600:    true,
		})
	}
	return plan, nil
}

type redactRule struct {
	id string
	re *regexp.Regexp
}

func compileRedactionRegexes() ([]redactRule, error) {
	suite, err := detect.Default()
	if err != nil {
		return nil, err
	}
	_ = suite
	// Use the same canonical patterns from detect to replace matched secrets with
	// [REDACTED:<kind>].
	specs := []struct {
		id  string
		pat string
	}{
		{"github-pat-classic", `\bghp_[A-Za-z0-9]{36}\b`},
		{"github-pat-fine", `\bgithub_pat_[A-Za-z0-9]{22}_[A-Za-z0-9]{59}\b`},
		{"github-oauth", `\bgho_[A-Za-z0-9]{36}\b`},
		{"aws-access-key-id", `\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`},
		{"npm-token", `\bnpm_[A-Za-z0-9]{36}\b`},
		{"pypi-token", `\bpypi-AgEIcHlwaS5vcmc[A-Za-z0-9\-_]{50,}\b`},
		{"anthropic-api-key", `\bsk-ant-api03-[A-Za-z0-9\-_]{32,}\b`},
		{"openai-api-key", `\bsk-(?:proj-)?[A-Za-z0-9\-_]{32,}\b`},
		{"stripe-live-key", `\b(?:sk|rk)_live_[A-Za-z0-9]{24,}\b`},
		{"slack-token", `\bxox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[A-Za-z0-9]{24,}\b`},
		{"gcp-api-key", `\bAIza[0-9A-Za-z\-_]{35}\b`},
	}
	out := make([]redactRule, 0, len(specs))
	for _, sp := range specs {
		re, err := regexp.Compile(sp.pat)
		if err != nil {
			return nil, err
		}
		out = append(out, redactRule{id: sp.id, re: re})
	}
	return out, nil
}

func redactBytes(in []byte, rules []redactRule) ([]byte, int) {
	text := string(in)
	total := 0
	for _, r := range rules {
		matches := r.re.FindAllString(text, -1)
		if len(matches) == 0 {
			continue
		}
		total += len(matches)
		text = r.re.ReplaceAllString(text, "[REDACTED:"+r.id+"]")
	}
	return []byte(text), total
}

// GitCredentialHelperFixer switches credential.helper = store to the OS
// keychain helper (osxkeychain / libsecret / manager).
type GitCredentialHelperFixer struct{}

func (GitCredentialHelperFixer) ID() string { return "git-credential-helper" }

func (GitCredentialHelperFixer) Plan(e env.Env, findings []probes.Finding, _ Options) (Plan, error) {
	targetHelper := "libsecret"
	switch e.OS() {
	case "darwin":
		targetHelper = "osxkeychain"
	case "windows":
		targetHelper = "manager"
	}

	var plan Plan
	seen := map[string]bool{}
	for _, f := range findings {
		if f.Kind != "git-credential-helper-store" {
			continue
		}
		rel := normalizeRel(f.Location.Path)
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		absPath := filepath.Join(e.Home(), filepath.FromSlash(rel))
		orig, err := os.ReadFile(filepath.Clean(absPath)) // #nosec G304 -- validated relative path inside Home
		if err != nil {
			continue
		}
		updated := replaceGitHelperStore(string(orig), targetHelper)
		if updated == string(orig) {
			continue
		}
		plan.Changes = append(plan.Changes, FileChange{
			FixerID:     "git-credential-helper",
			RelPath:     rel,
			ShortPath:   "~/" + rel,
			Description: fmt.Sprintf("credential.helper: store -> %s", targetHelper),
			NewContent:  []byte(updated),
			Chmod600:    true,
		})
	}
	return plan, nil
}

func replaceGitHelperStore(content, helper string) string {
	re := regexp.MustCompile(`(?im)^(\s*helper\s*=\s*)store(\s*)$`)
	return re.ReplaceAllString(content, "${1}"+helper+"${2}")
}

// NPMIgnoreScriptsFixer sets ignore-scripts=true in ~/.npmrc.
type NPMIgnoreScriptsFixer struct{}

func (NPMIgnoreScriptsFixer) ID() string { return "npm-ignore-scripts" }

func (NPMIgnoreScriptsFixer) Plan(e env.Env, findings []probes.Finding, _ Options) (Plan, error) {
	var plan Plan
	seen := map[string]bool{}
	for _, f := range findings {
		if f.Kind != "npm-ignore-scripts-disabled" {
			continue
		}
		rel := normalizeRel(f.Location.Path)
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		absPath := filepath.Join(e.Home(), filepath.FromSlash(rel))
		orig, err := os.ReadFile(filepath.Clean(absPath)) // #nosec G304 -- validated relative path inside Home
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?im)^(\s*ignore-scripts\s*=\s*)false(\s*)$`)
		updated := re.ReplaceAllString(string(orig), "${1}true${2}")
		if updated == string(orig) {
			continue
		}
		plan.Changes = append(plan.Changes, FileChange{
			FixerID:     "npm-ignore-scripts",
			RelPath:     rel,
			ShortPath:   "~/" + rel,
			Description: "set ignore-scripts=true (note: some packages require install scripts)",
			NewContent:  []byte(updated),
			Chmod600:    true,
		})
	}
	return plan, nil
}

// SSHDisableForwardAgentFixer changes ForwardAgent yes to ForwardAgent no in
// ~/.ssh/config.
type SSHDisableForwardAgentFixer struct{}

func (SSHDisableForwardAgentFixer) ID() string { return "ssh-disable-forward-agent" }

func (SSHDisableForwardAgentFixer) Plan(e env.Env, findings []probes.Finding, _ Options) (Plan, error) {
	var plan Plan
	seen := map[string]bool{}
	for _, f := range findings {
		if f.Kind != "ssh-forward-agent" {
			continue
		}
		rel := normalizeRel(f.Location.Path)
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		absPath := filepath.Join(e.Home(), filepath.FromSlash(rel))
		orig, err := os.ReadFile(filepath.Clean(absPath)) // #nosec G304 -- validated relative path inside Home
		if err != nil {
			continue
		}
		re := regexp.MustCompile(`(?im)^(\s*ForwardAgent(?:\s+|\s*=\s*))yes(\s*)$`)
		updated := re.ReplaceAllString(string(orig), "${1}no${2}")
		if updated == string(orig) {
			continue
		}
		plan.Changes = append(plan.Changes, FileChange{
			FixerID:     "ssh-disable-forward-agent",
			RelPath:     rel,
			ShortPath:   "~/" + rel,
			Description: "set ForwardAgent no in ~/.ssh/config",
			NewContent:  []byte(updated),
			Chmod600:    true,
		})
	}
	return plan, nil
}

// PermissionsFixer tightens file permissions to 0600 on credential files and
// SSH private keys.
type PermissionsFixer struct {
	id string
}

func (p PermissionsFixer) ID() string { return p.id }

func (p PermissionsFixer) Plan(e env.Env, findings []probes.Finding, _ Options) (Plan, error) {
	var plan Plan
	seen := map[string]bool{}
	for _, f := range findings {
		matched := false
		for _, fx := range f.Fixes {
			if fx == p.id {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		rel := normalizeRel(f.Location.Path)
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true
		absPath := filepath.Join(e.Home(), filepath.FromSlash(rel))
		if _, err := os.Stat(absPath); err != nil {
			continue
		}
		plan.Changes = append(plan.Changes, FileChange{
			FixerID:     p.id,
			RelPath:     rel,
			ShortPath:   "~/" + rel,
			Description: "chmod 0600 (owner read/write only)",
			Chmod600:    true,
		})
	}
	return plan, nil
}
