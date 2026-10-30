package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/report"
)

// ErrStrictFindings is returned when --strict is set and at least one finding exists.
var ErrStrictFindings = errors.New("findings detected under --strict mode")

func newScanCmd(o Options) *cobra.Command {
	var format string
	var probeFilter string
	var rootsFlag string
	var outputPath string
	var minSeverity string
	var noColor bool
	var strict bool

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan this computer for exposed credentials and build a loot map",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch format {
			case "pretty", "json", "markdown":
			default:
				return fmt.Errorf("unsupported --format %q (use pretty, json, or markdown)", format)
			}

			suite, err := detect.Default()
			if err != nil {
				return err
			}

			var only []string
			if probeFilter != "" {
				only = strings.Split(probeFilter, ",")
			}
			var roots []string
			if rootsFlag != "" {
				roots = strings.Split(rootsFlag, ",")
			}

			res, err := engine.Run(cmd.Context(), o.Env, o.Probes, suite, engine.Options{
				Version:    o.Version,
				OnlyProbes: only,
				Roots:      roots,
			})
			if err != nil {
				return err
			}

			if minSeverity != "" {
				res.Loot = filterByMinSeverity(res.Loot, minSeverity)
				res.Score = engine.CalculateScore(res.Loot)
			}

			var buf bytes.Buffer
			switch format {
			case "json":
				err = report.WriteJSON(&buf, res)
			case "markdown":
				err = report.WriteMarkdown(&buf, res)
			default:
				err = report.WritePretty(&buf, res, report.PrettyOptions{
					NoColor:     noColor,
					ProbesCount: len(o.Probes.All()),
					Width:       terminalWidth(o.Env.Getenv("COLUMNS")),
				})
			}
			if err != nil {
				return err
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, buf.Bytes(), 0o600); err != nil {
					return err
				}
			} else if _, err := cmd.OutOrStdout().Write(buf.Bytes()); err != nil {
				return err
			}

			if strict && len(res.Loot) > 0 {
				return ErrStrictFindings
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "pretty", "output format (pretty|json|markdown)")
	cmd.Flags().StringVar(&probeFilter, "probe", "", "comma-separated probe IDs to run")
	cmd.Flags().StringVar(&rootsFlag, "roots", "", "comma-separated development roots to scan (e.g. ~/code,~/work)")
	cmd.Flags().StringVar(&outputPath, "output", "", "write report to file instead of stdout")
	cmd.Flags().StringVar(&minSeverity, "min-severity", "", "minimum severity to report (low|medium|high|critical)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "disable ANSI colors in pretty output")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit with code 1 if any findings are reported")
	return cmd
}

func filterByMinSeverity(items []engine.LootItem, minSev string) []engine.LootItem {
	thresh := sevLevel(strings.ToLower(strings.TrimSpace(minSev)))
	if thresh == 0 {
		return items
	}
	var out []engine.LootItem
	for _, it := range items {
		if sevLevel(it.Severity) >= thresh {
			out = append(out, it)
		}
	}
	return out
}

func sevLevel(s string) int {
	switch s {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}

// terminalWidth parses $COLUMNS; anything unparsable means the 80-column
// default.
func terminalWidth(columns string) int {
	n, err := strconv.Atoi(strings.TrimSpace(columns))
	if err != nil {
		return 80
	}
	return n
}
