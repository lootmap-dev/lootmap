package probes

import (
	"context"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// ShellHistoryProbe scans bash, zsh, fish, and PowerShell history files.
type ShellHistoryProbe struct{}

func (ShellHistoryProbe) ID() string               { return "shell-history" }
func (ShellHistoryProbe) Available(e env.Env) bool { return AnyPathExists(e, "shell-history") }

func (ShellHistoryProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	for _, rel := range PathsFor("shell-history", e.OS()) {
		_ = ScanTextFile(e, "shell-history", rel, emit)
	}
	return nil
}
