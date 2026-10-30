package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/loot"
)

func newExplainCmd(o Options) *cobra.Command {
	var rotateOnly bool

	cmd := &cobra.Command{
		Use:   "explain [finding-id-or-kind]",
		Short: "Explain a finding and how to remediate or rotate it",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if rotateOnly {
				return renderRotateGuide(cmd.OutOrStdout(), res)
			}
			if len(args) == 0 {
				return fmt.Errorf("specify a finding ID (e.g. L-1234abcd), kind, or --rotate")
			}
			return renderExplainTarget(cmd.OutOrStdout(), res, args[0])
		},
	}

	cmd.Flags().BoolVar(&rotateOnly, "rotate", false, "list rotation links and steps for all found secrets")
	return cmd
}

func renderRotateGuide(w io.Writer, res engine.Result) error {
	cat := loot.Default()
	count := 0
	for _, it := range res.Loot {
		if it.Rotate == "" {
			continue
		}
		count++
		ent, _ := cat.Lookup(it.Kind)
		if _, err := fmt.Fprintf(w, "%s (%s) · %s\n  Rotate: %s\n", it.ID, it.Kind, it.Preview, it.Rotate); err != nil {
			return err
		}
		for _, step := range ent.Steps {
			if _, err := fmt.Fprintf(w, "  - %s\n", step); err != nil {
				return err
			}
		}
	}
	if count == 0 {
		_, err := fmt.Fprintln(w, "No keys requiring rotation were found.")
		return err
	}
	return nil
}

func renderExplainTarget(w io.Writer, res engine.Result, target string) error {
	cat := loot.Default()
	for _, it := range res.Loot {
		if it.ID == target || it.Kind == target {
			ent, _ := cat.Lookup(it.Kind)
			if _, err := fmt.Fprintf(w, "%s · %s (%s)\nImpact: %s\nPreview: %s\n",
				it.ID, ent.Title, it.Severity, it.Impact, it.Preview); err != nil {
				return err
			}
			if it.Rotate != "" {
				if _, err := fmt.Fprintf(w, "Rotate: %s\n", it.Rotate); err != nil {
					return err
				}
			}
			for _, step := range ent.Steps {
				if _, err := fmt.Fprintf(w, "- %s\n", step); err != nil {
					return err
				}
			}
			return nil
		}
	}
	if ent, ok := cat.Lookup(target); ok {
		if _, err := fmt.Fprintf(w, "%s (%s)\nImpact: %s\n", ent.Title, ent.Severity, ent.Impact); err != nil {
			return err
		}
		if ent.Rotate != "" {
			if _, err := fmt.Fprintf(w, "Rotate: %s\n", ent.Rotate); err != nil {
				return err
			}
		}
		for _, step := range ent.Steps {
			if _, err := fmt.Fprintf(w, "- %s\n", step); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("finding %q not found", target)
}
