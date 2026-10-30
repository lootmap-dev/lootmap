package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/loot"
)

// WriteMarkdown renders Result as a shareable, secret-free Markdown report.
func WriteMarkdown(w io.Writer, res engine.Result) error {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "# lootmap report (%s)\n\n", res.Host.OS)
	_, _ = fmt.Fprintf(&b, "**Score:** %d/100 (%s) · **Items:** %d\n\n",
		res.Score, loot.Grade(res.Score), len(res.Loot))

	if len(res.Loot) == 0 {
		b.WriteString("No exposed credentials or risky configurations found.\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	b.WriteString("| ID | Severity | Kind | Preview | Impact | Locations |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, it := range res.Loot {
		var locs []string
		for _, l := range it.Locations {
			if l.Line > 0 {
				locs = append(locs, fmt.Sprintf("`%s:%d`", l.Path, l.Line))
			} else {
				locs = append(locs, fmt.Sprintf("`%s`", l.Path))
			}
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` | %s | %s |\n",
			it.ID, it.Severity, it.Kind, it.Preview, it.Impact, strings.Join(locs, ", "))
	}
	_, err := io.WriteString(w, b.String())
	return err
}
