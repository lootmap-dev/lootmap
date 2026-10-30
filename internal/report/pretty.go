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
	// Width is the terminal width in columns. Values below 80 (including the
	// zero value) render at 80; values above 120 render at 120.
	Width int
}

// WritePretty renders the loot map summary screen (SPEC.md CLI design). It
// never exceeds opts.Width columns (80 by default).
func WritePretty(w io.Writer, res engine.Result, opts PrettyOptions) error {
	cat := loot.Default()
	width := opts.Width
	if width < 80 {
		width = 80
	}
	if width > 120 {
		width = 120
	}
	fit := func(line string) string { return truncateTo(line, width) }
	// Column budget: "  ● " (4) + title + " " + desc + " " + location.
	titleW := 18
	descW := 30 + (width-80)/2
	locW := width - 4 - titleW - 1 - descW - 1

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
	probesWord := "probes"
	if probesCount == 1 {
		probesWord = "probe"
	}
	header := fmt.Sprintf("lootmap %s · %s · %d %s · network: NONE",
		res.Version, res.Host.OS, probesCount, probesWord)
	b.WriteString(fit(header) + "\n\n")

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
		if autoFixable(it) {
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

		title := padOrTrim(ent.Title, titleW)
		desc := ent.Short
		if len(it.Locations) > 1 {
			desc = fmt.Sprintf("%s · %d places", ent.Short, len(it.Locations))
		}
		desc = padOrTrim(desc, descW)

		locStr := ""
		if len(it.Locations) > 0 {
			locStr = it.Locations[0].Path
			if len(it.Locations) > 1 {
				more := fmt.Sprintf(" +%d", len(it.Locations)-1)
				locStr = shortenPath(locStr, locW-len(more)) + more
			} else {
				locStr = shortenPath(locStr, locW)
			}
		}

		rawLine := fmt.Sprintf("  %s %s %s %s", bullet, title, desc, locStr)
		b.WriteString(fit(rawLine) + "\n")
	}

	b.WriteString("\n")
	grade := loot.Grade(res.Score)
	scoreText := fmt.Sprintf("Score: %d/100 (%s)", res.Score, grade)
	if !opts.NoColor {
		gradeColor := "9" // red
		switch grade {
		case "A", "B":
			gradeColor = "10" // green
		case "C", "D":
			gradeColor = "11" // yellow
		}
		scoreText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(gradeColor)).Render(scoreText)
	}
	scoreLine := fmt.Sprintf(" %s · %d critical · %d high · %d medium",
		scoreText, critCount, highCount, medCount)
	if lowCount > 0 {
		scoreLine += fmt.Sprintf(" · %d low", lowCount)
	}
	b.WriteString(fit(scoreLine) + "\n\n")

	if fixableCount > 0 {
		b.WriteString(fit(fmt.Sprintf(" %-24s -> lootmap fix", fmt.Sprintf("%d auto-fixable", fixableCount))) + "\n")
	}
	if rotateCount > 0 {
		keysPhrase := fmt.Sprintf("%d keys need rotation", rotateCount)
		if rotateCount == 1 {
			keysPhrase = "1 key needs rotation"
		}
		b.WriteString(fit(fmt.Sprintf(" %-24s -> lootmap explain --rotate", keysPhrase)) + "\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func padOrTrim(s string, width int) string {
	if ansi.StringWidth(s) > width {
		s = ansi.Truncate(s, width, "…")
	}
	return s + strings.Repeat(" ", width-ansi.StringWidth(s))
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

// shortenPath keeps the first directory and the file name, which are the two
// parts a reader needs to recognise a location:
// "~/.claude/projects/acme-web/9f2c1e.jsonl" -> "~/.claude/…/9f2c1e.jsonl".
func shortenPath(p string, max int) string {
	if ansi.StringWidth(p) <= max {
		return p
	}
	parts := strings.Split(p, "/")
	if len(parts) >= 3 {
		head := parts[0] + "/" + parts[1]
		tail := parts[len(parts)-1]
		if cand := head + "/…/" + tail; ansi.StringWidth(cand) <= max {
			return cand
		}
		if cand := "…/" + tail; ansi.StringWidth(cand) <= max {
			return cand
		}
	}
	return shortenMiddle(p, max)
}

func truncateTo(line string, width int) string {
	if ansi.StringWidth(line) <= width {
		return strings.TrimRight(line, " ")
	}
	return ansi.Truncate(line, width, "…")
}

// autoFixable reports whether `lootmap fix` can actually remove this item:
// secrets that leaked into AI transcripts or shell history can be redacted,
// and a few configuration findings can be corrected in place. Secrets that
// live in their tool's own credential file must be rotated instead, so they
// are not counted even when a hardening fix (such as chmod 600) applies.
func autoFixable(it engine.LootItem) bool {
	switch it.Kind {
	case "git-credential-helper-store", "npm-ignore-scripts-disabled",
		"ssh-key-permissions", "ssh-forward-agent":
		return true
	}
	for _, l := range it.Locations {
		if l.Probe == "ai-transcripts" || l.Probe == "shell-history" {
			return true
		}
	}
	return false
}
