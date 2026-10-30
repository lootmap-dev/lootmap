package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/osv"
	"github.com/lootmap-dev/lootmap/internal/report"
)

func newCheckPackagesCmd(o Options) *cobra.Command {
	var format string
	var rootsFlag string
	var strict bool

	cmd := &cobra.Command{
		Use:   "check-packages",
		Short: "Check lockfiles and package caches for known malicious OSV MAL- versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			suite, err := detect.Default()
			if err != nil {
				return err
			}
			var roots []string
			if rootsFlag != "" {
				roots = strings.Split(rootsFlag, ",")
			}

			res, err := engine.Run(cmd.Context(), o.Env, o.Probes, suite, engine.Options{
				Version:    o.Version,
				OnlyProbes: []string{"packages"},
				Roots:      roots,
			})
			if err != nil {
				return err
			}

			switch format {
			case "json":
				if err := report.WriteJSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			case "markdown":
				if err := report.WriteMarkdown(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			default:
				if err := report.WritePretty(cmd.OutOrStdout(), res, report.PrettyOptions{
					NoColor:     true,
					ProbesCount: 1,
				}); err != nil {
					return err
				}
			}

			if strict && len(res.Loot) > 0 {
				return ErrStrictFindings
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "pretty", "output format (pretty|json|markdown)")
	cmd.Flags().StringVar(&rootsFlag, "roots", "", "comma-separated development roots to scan")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit with code 1 if malicious packages are found")
	return cmd
}

func newUpdateDBCmd(o Options) *cobra.Command {
	var url string

	cmd := &cobra.Command{
		Use:   "update-db",
		Short: "Download and verify the latest signed OSV MAL- advisory bundle (uses network)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			db, err := osv.DownloadAndUpdate(cmd.Context(), osv.UpdateOptions{
				Home: o.Env.Home(),
				URL:  url,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated OSV MAL- database to version %s.\n", db.Version)
			return err
		},
	}

	cmd.Flags().StringVar(&url, "url", osv.DefaultBundleURL, "URL of the signed OSV MAL- bundle")
	return cmd
}
