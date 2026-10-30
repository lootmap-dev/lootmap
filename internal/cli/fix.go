package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/fix"
	"github.com/lootmap-dev/lootmap/internal/probes"
)

func newFixCmd(o Options) *cobra.Command {
	var dryRun bool
	var yes bool
	var force bool

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Preview, back up, and apply safe remediations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			suite, err := detect.Default()
			if err != nil {
				return err
			}
			res, err := engine.Run(cmd.Context(), o.Env, o.Probes, suite, engine.Options{
				Version: o.Version,
			})
			if err != nil {
				return err
			}

			findings := flattenLootToFindings(res.Loot)
			fixOpts := fix.Options{
				Force: force,
				Now:   time.Now,
			}
			plan, err := fix.BuildPlan(o.Env, findings, fixOpts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, sk := range plan.Skipped {
				_, _ = fmt.Fprintf(out, "skipped (live session protection): %s\n", sk)
			}
			if len(plan.Changes) == 0 {
				_, _ = fmt.Fprintln(out, "No automatic fixes to apply.")
				return nil
			}

			_, _ = fmt.Fprintf(out, "Planned changes (%d):\n\n", len(plan.Changes))
			pathW := 0
			for _, ch := range plan.Changes {
				if n := len([]rune(ch.ShortPath)); n > pathW {
					pathW = n
				}
			}
			for _, ch := range plan.Changes {
				_, _ = fmt.Fprintf(out, "  • %-*s  %s\n", pathW, ch.ShortPath, ch.Description)
			}
			_, _ = fmt.Fprintln(out)

			if dryRun {
				_, _ = fmt.Fprintln(out, "Dry run complete; no files were modified.")
				return nil
			}

			if !yes {
				_, _ = fmt.Fprintln(out, "Re-run with --yes to apply these changes (backups are saved for 7 days).")
				return nil
			}

			bmgr := fix.NewBackupManager(o.Env.Home())
			manifest, err := fix.Apply(o.Env, plan, bmgr, fixOpts)
			if err != nil {
				return err
			}
			word := "fixes"
			if len(plan.Changes) == 1 {
				word = "fix"
			}
			_, _ = fmt.Fprintf(out, "Applied %d %s. Originals backed up to ~/.lootmap/backups/%s\n", len(plan.Changes), word, manifest.ID)
			_, _ = fmt.Fprintln(out, "Backups are deleted after 7 days. Run 'lootmap undo' to revert.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview planned changes without modifying any files")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm and apply all planned fixes")
	cmd.Flags().BoolVar(&force, "force", false, "override 60-second live session protection on AI transcripts")
	return cmd
}

func newUndoCmd(o Options) *cobra.Command {
	return &cobra.Command{
		Use:   "undo [backup-id]",
		Short: "Restore files from the most recent or specified backup snapshot",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			bmgr := fix.NewBackupManager(o.Env.Home())
			manifest, err := bmgr.Restore(id)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Restored %d file(s) from backup %s.\n", len(manifest.Entries), manifest.ID)
			return err
		},
	}
}

func flattenLootToFindings(lootItems []engine.LootItem) []probes.Finding {
	var out []probes.Finding
	for _, it := range lootItems {
		for _, loc := range it.Locations {
			out = append(out, probes.Finding{
				Kind:        it.Kind,
				Impact:      it.Impact,
				Severity:    it.Severity,
				Fingerprint: it.Fingerprint,
				Preview:     it.Preview,
				Location:    loc,
				Fixes:       it.Fixes,
				Rotate:      it.Rotate,
			})
		}
	}
	return out
}
