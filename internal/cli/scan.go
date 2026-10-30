package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/report"
)

func newScanCmd(o Options) *cobra.Command {
	var format string
	var probeFilter string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan this computer for exposed credentials and build a loot map",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch format {
			case "json", "pretty":
			default:
				return fmt.Errorf("unsupported --format %q (use json or pretty)", format)
			}

			suite, err := detect.Default()
			if err != nil {
				return err
			}

			var only []string
			if probeFilter != "" {
				only = strings.Split(probeFilter, ",")
			}

			res, err := engine.Run(cmd.Context(), o.Env, o.Probes, suite, engine.Options{
				Version:    o.Version,
				OnlyProbes: only,
			})
			if err != nil {
				return err
			}

			// Phase 1 implements JSON output; Phase 5 replaces the pretty fallback
			// with the lipgloss loot map report.
			return report.WriteJSON(cmd.OutOrStdout(), res)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "output format (json|pretty)")
	cmd.Flags().StringVar(&probeFilter, "probe", "", "comma-separated probe IDs to run")
	return cmd
}
