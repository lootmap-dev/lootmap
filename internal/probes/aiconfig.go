package probes

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// AIConfigProbe scans AI agent and MCP configuration files (~/.claude.json,
// claude_desktop_config.json, ~/.codex/config.toml, ~/.cursor/mcp.json,
// Windsurf and VS Code MCP configs) for hardcoded tokens.
type AIConfigProbe struct{}

func (AIConfigProbe) ID() string               { return "ai-config" }
func (AIConfigProbe) Available(e env.Env) bool { return AnyPathExists(e, "ai-config") }

func (AIConfigProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	for _, rel := range PathsFor("ai-config", e.OS()) {
		data, err := fs.ReadFile(fsys, rel)
		if err != nil {
			continue
		}
		if strings.ToLower(path.Ext(rel)) == ".toml" {
			scanTOMLConfig(rel, data, emit)
		}
		_ = ScanTextFile(e, "ai-config", rel, emit)
	}
	return nil
}

func scanTOMLConfig(rel string, data []byte, emit Emitter) {
	var parsed map[string]any
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return
	}
	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	flattenTOML(shortPath, parsed, emit)
}

func flattenTOML(shortPath string, m map[string]any, emit Emitter) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			emit.EmitChunk(Chunk{
				Probe: "ai-config",
				Path:  shortPath,
				Line:  1,
				Text:  fmt.Sprintf("%s = %q", k, val),
			})
		case map[string]any:
			flattenTOML(shortPath, val, emit)
		}
	}
}
