// Package detect compiles the embedded patterns.yaml rules and scans Chunks
// for secrets while filtering out placeholders, example files, and low-entropy
// strings.
package detect

import (
	_ "embed"
	"math"
	"net/url"
	"path"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/secret"
)

//go:embed patterns.yaml
var embeddedPatterns []byte

// Detector recognizes one secret kind in a Chunk.
type Detector interface {
	ID() string
	Find(c probes.Chunk) []secret.Match
}

// RuleMeta holds the catalog/metadata fields associated with a pattern ID.
type RuleMeta struct {
	ID       string
	Severity string
	Impact   string
	Fixes    []string
	Rotate   string
}

type patternSpec struct {
	ID         string   `yaml:"id"`
	Regex      string   `yaml:"regex"`
	PrefixLen  int      `yaml:"prefix_len"`
	SuffixLen  int      `yaml:"suffix_len"`
	MinEntropy float64  `yaml:"min_entropy"`
	Severity   string   `yaml:"severity"`
	Impact     string   `yaml:"impact"`
	Fixes      []string `yaml:"fixes"`
	Rotate     string   `yaml:"rotate"`
}

type patternsFile struct {
	Patterns []patternSpec `yaml:"patterns"`
}

type compiledDetector struct {
	spec patternSpec
	re   *regexp.Regexp
}

func (d *compiledDetector) ID() string { return d.spec.ID }

func (d *compiledDetector) Find(c probes.Chunk) []secret.Match {
	if isIgnoredPath(c.Path) {
		return nil
	}
	if strings.Contains(c.Text, "data:image/") && strings.Contains(c.Text, ";base64,") {
		return nil
	}

	// openai-api-key must not match anthropic keys (sk-ant-...).
	all := d.re.FindAllStringSubmatchIndex(c.Text, -1)
	if len(all) == 0 {
		return nil
	}

	var out []secret.Match
	for _, loc := range all {
		start, end := loc[0], loc[1]
		if len(loc) >= 4 && loc[2] >= 0 && loc[3] >= 0 {
			start, end = loc[2], loc[3]
		}
		candidate := c.Text[start:end]
		if d.spec.ID == "openai-api-key" && strings.HasPrefix(candidate, "sk-ant-") {
			continue
		}
		if isFalsePositive(d.spec.ID, candidate) {
			continue
		}
		if d.spec.MinEntropy > 0 && shannonEntropy(entropySubject(d.spec.ID, candidate)) < d.spec.MinEntropy {
			continue
		}
		line := c.Line
		if line <= 0 {
			line = 1
		}
		out = append(out, secret.Match{
			Kind:  d.spec.ID,
			Value: secret.NewValue(candidate, d.spec.PrefixLen, d.spec.SuffixLen),
			Line:  line,
		})
	}
	return out
}

// Suite holds all compiled detectors and their metadata.
type Suite struct {
	detectors []Detector
	meta      map[string]RuleMeta
}

// Default parses and compiles the embedded patterns.yaml rules.
func Default() (*Suite, error) {
	return Parse(embeddedPatterns)
}

// Parse compiles a YAML pattern definition document into a Suite.
func Parse(data []byte) (*Suite, error) {
	var pf patternsFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	s := &Suite{
		detectors: make([]Detector, 0, len(pf.Patterns)),
		meta:      make(map[string]RuleMeta, len(pf.Patterns)),
	}
	for _, sp := range pf.Patterns {
		re, err := regexp.Compile(sp.Regex)
		if err != nil {
			return nil, err
		}
		s.detectors = append(s.detectors, &compiledDetector{spec: sp, re: re})
		s.meta[sp.ID] = RuleMeta{
			ID:       sp.ID,
			Severity: sp.Severity,
			Impact:   sp.Impact,
			Fixes:    sp.Fixes,
			Rotate:   sp.Rotate,
		}
	}
	return s, nil
}

// Detectors returns the ordered slice of Detectors in the suite.
func (s *Suite) Detectors() []Detector { return s.detectors }

// Meta returns the RuleMeta for a detector ID.
func (s *Suite) Meta(kind string) (RuleMeta, bool) {
	m, ok := s.meta[kind]
	return m, ok
}

// FindAll runs every detector in the suite on c.
func (s *Suite) FindAll(c probes.Chunk) []secret.Match {
	if isIgnoredPath(c.Path) {
		return nil
	}
	var matches []secret.Match
	for _, d := range s.detectors {
		matches = append(matches, d.Find(c)...)
	}
	return matches
}

func isIgnoredPath(p string) bool {
	base := strings.ToLower(path.Base(strings.ReplaceAll(p, `\`, "/")))
	switch base {
	case ".env.example", ".env.sample", ".env.template", ".env.dist":
		return true
	}
	return strings.HasSuffix(base, ".env.example") ||
		strings.HasSuffix(base, ".env.sample") ||
		strings.HasSuffix(base, ".env.template")
}

var placeholderTokens = []string{
	"akiai" + "osfodnn7example", // AWS official example key
	"wjalr" + "XUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	"changeme",
	"your_api_key",
	"your-api-key",
	"replace_me",
	"replaceme",
	"todo",
	"example",
	"placeholder",
	"[redacted",
}

func isFalsePositive(kind, val string) bool {
	lower := strings.ToLower(val)
	if strings.Contains(val, "${") || strings.HasPrefix(val, "$") {
		return true
	}
	for _, tok := range placeholderTokens {
		if strings.Contains(lower, strings.ToLower(tok)) {
			return true
		}
	}
	if hasRepeatedRun(lower, 8) {
		return true
	}
	if kind == "database-url" {
		return isPlaceholderDatabaseURL(val)
	}
	return false
}

func isPlaceholderDatabaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return true
	}
	pw, ok := u.User.Password()
	if !ok || pw == "" {
		return true
	}
	if strings.Contains(pw, "${") || strings.HasPrefix(pw, "$") {
		return true
	}
	low := strings.ToLower(pw)
	for _, tok := range []string{"changeme", "password", "secret", "example", "xxx", "todo", "postgres", "root", "admin"} {
		if low == tok || strings.Contains(low, "changeme") || strings.Contains(low, "replace") {
			return true
		}
	}
	return false
}

func hasRepeatedRun(s string, runLen int) bool {
	if len(s) < runLen {
		return false
	}
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= runLen {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

func entropySubject(kind, val string) string {
	if kind == "private-key-pem" {
		lines := strings.Split(val, "\n")
		if len(lines) > 2 {
			return strings.Join(lines[1:len(lines)-1], "")
		}
	}
	if kind == "database-url" {
		if u, err := url.Parse(val); err == nil && u.User != nil {
			if pw, ok := u.User.Password(); ok {
				return pw
			}
		}
	}
	// Strip common short prefixes before measuring entropy.
	for _, pfx := range []string{"github_pat_", "sk-ant-api03-", "sk-proj-", "sk_live_", "rk_live_", "ghp_", "gho_", "npm_", "AIza"} {
		if strings.HasPrefix(val, pfx) && len(val) > len(pfx) {
			return val[len(pfx):]
		}
	}
	return val
}

func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	var counts [256]int
	for i := 0; i < len(s); i++ {
		counts[s[i]]++
	}
	n := float64(len(s))
	var ent float64
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		ent -= p * math.Log2(p)
	}
	return ent
}
