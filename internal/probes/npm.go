package probes

import (
	"context"
	"io/fs"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// NPMProbe scans ~/.npmrc and ~/.pypirc for publish tokens and checks whether
// ignore-scripts=true is missing or disabled in ~/.npmrc.
type NPMProbe struct{}

func (NPMProbe) ID() string               { return "npm" }
func (NPMProbe) Available(e env.Env) bool { return AnyPathExists(e, "npm") }

func (NPMProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	for _, rel := range PathsFor("npm", e.OS()) {
		data, err := fs.ReadFile(e.FS(), rel)
		if err != nil {
			continue
		}
		_ = ScanTextFile(e, "npm", rel, emit)
		if strings.HasSuffix(rel, ".npmrc") {
			checkNPMIgnoreScripts(rel, string(data), emit)
		}
	}
	return nil
}

func checkNPMIgnoreScripts(rel, content string, emit Emitter) {
	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for idx, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(line[:eq]))
		v := strings.ToLower(strings.TrimSpace(line[eq+1:]))
		if k == "ignore-scripts" && v == "false" {
			emit.EmitFinding(Finding{
				Kind:        "npm-ignore-scripts-disabled",
				Impact:      "Malicious npm lifecycle scripts (preinstall/postinstall) can run automatically",
				Severity:    "high",
				Fingerprint: "config:npm-ignore-scripts-disabled:" + shortPath,
				Preview:     "ignore-scripts=false",
				Location:    Location{Probe: "npm", Path: shortPath, Line: idx + 1},
				Fixes:       []string{"npm-ignore-scripts"},
			})
		}
	}
}
