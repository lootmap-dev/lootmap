// Package cli wires the cobra command tree. Commands receive their Env,
// probe registry and output streams through Options, so tests run them
// against fixture homes without touching the real system.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
)

// Options carries every external dependency of the command tree.
type Options struct {
	Env     env.Env
	Probes  *probes.Registry
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

// NewRootCmd builds the lootmap command tree.
func NewRootCmd(o Options) *cobra.Command {
	root := &cobra.Command{
		Use:   "lootmap",
		Short: "See your laptop the way malware does. Then fix it.",
		Long: "lootmap shows what an infostealer or a Shai-Hulud style worm would take\n" +
			"from this computer, and helps you lock it down.\n\n" +
			"lootmap never uses the network by default and never prints secret values.",
		Version:       o.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetVersionTemplate("lootmap version {{.Version}}\n")
	root.SetOut(o.Stdout)
	root.SetErr(o.Stderr)

	root.AddCommand(newDoctorCmd(o))
	root.AddCommand(newScanCmd(o))
	root.AddCommand(newExplainCmd(o))
	root.AddCommand(newFixCmd(o))
	root.AddCommand(newUndoCmd(o))
	return root
}

// Execute runs lootmap against the real system and returns the exit code.
func Execute(version string, args []string) int {
	e, err := env.System()
	if err != nil {
		fmt.Fprintln(os.Stderr, "lootmap:", err)
		return 2
	}
	cmd := NewRootCmd(Options{
		Env:     e,
		Probes:  probes.Default(),
		Version: version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "lootmap:", err)
		return 1
	}
	return 0
}
