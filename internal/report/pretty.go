package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/loot"
)

// PrettyOptions controls terminal rendering for WritePretty.
type PrettyOptions struct {
	NoColor     bool
	ProbesCount int
}

// WritePretty renders the 80-column loot map summary screen (SPEC.md CLI design).
func WritePretty(w io.Writer, res engine.Result, opts PrettyOptions) error {
	cat := loot.Default()

	bold := func(s string) string {
		if opts.NoColor {
			return s
		}
		return lipgloss.NewStyle().Bold(true).Render(s)
	}
	red := func(s string) string {
		if opts.NoColor {
			return s
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(s)
	}
	yellow := func(s string) string {
		if opts.NoColor {
			return s
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(s)
	}

	var b strings.Builder

	probesCount := opts.ProbesCount
	if probesCount <= 0 {
		probesCount = 12
	}
	header := fmt.Sprintf("lootmap %s · %s · %d probes · network: NONE",
		res.Version, res.Host.OS, probesCount)
	b.WriteString(truncate80(header) + "\n\n")

	if len(res.Loot) == 0 {
		b.WriteString(" " + bold("NO EXPOSED LOOT FOUND ON THIS COMPUTER.") + "\n\n")
		_, _ = fmt.Fprintf(&b, " Score: %d/100 (%s) · 0 critical · 0 high · 0 medium\n",
			res.Score, loot.Grade(res.Score))
		_, err := io.WriteString(w, b.String())
		return err
	}

	b.WriteString(" " + bold("IF MALWARE RAN ON THIS COMPUTER IT WOULD TAKE:") + "\n\n")

	var critCount, highCount, medCount, lowCount int
	var fixableCount, rotateCount int

	for _, it := range res.Loot {
		switch it.Severity {
		case "critical":
			critCount++
		case "high":
			highCount++
		case "medium":
			medCount++
		case "low":
			lowCount++
		}
		if len(it.Fixes) > 0 {
			fixableCount++
		}
		if it.Rotate != "" {
			rotateCount++
		}

		ent, _ := cat.Lookup(it.Kind)
		bullet := "○"
		switch it.Severity {
		case "critical":
			bullet = red("●")
		case "high":
			bullet = yellow("●")
		}

		title := padOrTrim(ent.Title, 18)
		desc := ent.Short
		if len(it.Locations) > 1 {
			desc = fmt.Sprintf("%s · %d places", ent.Short, len(it.Locations))
		}
		desc = padOrTrim(desc, 30)

		locStr := ""
		if len(it.Locations) > 0 {
			locStr = it.Locations[0].Path
			if len(it.Locations) > 1 {
				locStr = fmt.Sprintf("%s +%d", shortenMiddle(locStr, 18), len(it.Locations)-1)
			} else {
				locStr = shortenMiddle(locStr, 22)
			}
		}

		rawLine := fmt.Sprintf("  %s %s %s %s", bullet, title, desc, locStr)
		b.WriteString(truncate80(rawLine) + "\n")
	}

	b.WriteString("\n")
	scoreLine := fmt.Sprintf(" Score: %d/100 (%s) · %d critical · %d high · %d medium",
		res.Score, loot.Grade(res.Score), critCount, highCount, medCount)
	if lowCount > 0 {
		scoreLine += fmt.Sprintf(" · %d low", lowCount)
	}
	b.WriteString(truncate80(scoreLine) + "\n\n")

	if fixableCount > 0 {
		b.WriteString(truncate80(fmt.Sprintf(" %d findings auto-fixable -> lootmap fix", fixableCount)) + "\n")
	}
	if rotateCount > 0 {
		b.WriteString(truncate80(fmt.Sprintf(" %d keys need rotation    -> lootmap explain --rotate", rotateCount)) + "\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func padOrTrim(s string, width int) string {
	w := ansi.StringWidth(s)
	if w > width {
		return ansi.Truncate(s, width-1, "…")
	}
	return s + strings.Repeat(" ", width-w)
}

func shortenMiddle(p string, max int) string {
	if len(p) <= max {
		return p
	}
	if max <= 6 {
		return p[:max]
	}
	half := (max - 1) / 2
	return p[:half] + "…" + p[len(p)-(max-1-half):]
}

func truncate80(line string) string {
	if ansi.StringWidth(line) <= 80 {
		return strings.TrimRight(line, " ")
	}
	return ansi.Truncate(line, 80, "…")
}
