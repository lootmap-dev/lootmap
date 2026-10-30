// Package report formats scan results. Phase 1 provides the leak-free JSON
// reporter; pretty (lipgloss) and Markdown reporters are added in Phase 5.
package report

import (
	"encoding/json"
	"io"

	"github.com/lootmap-dev/lootmap/internal/engine"
)

// WriteJSON serializes Result as indented JSON to w.
func WriteJSON(w io.Writer, res engine.Result) error {
	if res.Loot == nil {
		res.Loot = []engine.LootItem{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(res)
}
