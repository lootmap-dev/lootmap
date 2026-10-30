package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newDoctorCmd(o Options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Show the environment and which probes would run",
		Long: "doctor reports the operating system, the home directory (shortened)\n" +
			"and every registered probe with whether it is available here.\n" +
			"It reads nothing but tool locations and never uses the network.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd.OutOrStdout(), o)
		},
	}
}

func runDoctor(w io.Writer, o Options) error {
	e := o.Env
	all := o.Probes.All()

	var err error
	p := func(format string, a ...any) {
		if err == nil {
			_, err = fmt.Fprintf(w, format, a...)
		}
	}
	p("lootmap %s · doctor\n\n", o.Version)
	p("  %-9s %s\n", "os", e.OS())
	p("  %-9s %s\n", "home", e.Shorten(e.Home()))
	p("  %-9s %s\n", "network", "NONE")
	p("  %-9s %d registered\n", "probes", len(all))
	if len(all) == 0 {
		p("  %-9s (none)\n", "")
		return err
	}
	width := 0
	for _, pr := range all {
		if n := len(pr.ID()); n > width {
			width = n
		}
	}
	for _, pr := range all {
		status := "not available"
		if pr.Available(e) {
			status = "available"
		}
		p("  %-9s %-*s   %s\n", "", width, pr.ID(), status)
	}
	return err
}
