package probes

import (
	"context"
	"io/fs"
	"path"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// GitProbe scans ~/.git-credentials, ~/.gitconfig, ~/.config/git/config (plus
// include.path targets), and ~/.config/gh/hosts.yml without running git.
type GitProbe struct{}

func (GitProbe) ID() string               { return "git" }
func (GitProbe) Available(e env.Env) bool { return AnyPathExists(e, "git") }

func (GitProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	visited := map[string]bool{}
	for _, rel := range PathsFor("git", e.OS()) {
		scanGitFile(e, rel, emit, visited, 0)
	}
	return nil
}

func scanGitFile(e env.Env, rel string, emit Emitter, visited map[string]bool, depth int) {
	if depth > 5 {
		return
	}
	rel = strings.TrimPrefix(path.Clean(strings.ReplaceAll(rel, `\`, "/")), "/")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "..") || visited[rel] {
		return
	}
	visited[rel] = true

	data, err := fs.ReadFile(e.FS(), rel)
	if err != nil {
		return
	}
	_ = ScanTextFile(e, "git", rel, emit)

	base := path.Base(rel)
	if base != ".gitconfig" && base != "config" && !strings.HasSuffix(base, ".gitconfig") {
		return
	}

	shortPath := "~/" + rel
	inCredential := false
	inInclude := false
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	for idx, rawLine := range lines {
		lineNo := idx + 1
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sec := strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			inCredential = sec == "credential" || strings.HasPrefix(sec, "credential ")
			inInclude = sec == "include" || strings.HasPrefix(sec, "includeif ")
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)

		if inCredential && key == "helper" && strings.EqualFold(val, "store") {
			emit.EmitFinding(Finding{
				Kind:        "git-credential-helper-store",
				Impact:      "Git credentials are stored in plaintext on disk",
				Severity:    "critical",
				Fingerprint: "config:git-credential-helper-store:" + shortPath,
				Preview:     "helper = store",
				Location:    Location{Probe: "git", Path: shortPath, Line: lineNo},
				Fixes:       []string{"git-credential-helper"},
			})
		}

		if inInclude && key == "path" && val != "" {
			incRel := resolveGitInclude(rel, val)
			if incRel != "" {
				scanGitFile(e, incRel, emit, visited, depth+1)
			}
		}
	}
}

func resolveGitInclude(currentRel, inc string) string {
	inc = strings.ReplaceAll(inc, `\`, "/")
	if strings.HasPrefix(inc, "~/") {
		return strings.TrimPrefix(inc, "~/")
	}
	if strings.HasPrefix(inc, "/") || (len(inc) >= 2 && inc[1] == ':') {
		return ""
	}
	return path.Join(path.Dir(currentRel), inc)
}
