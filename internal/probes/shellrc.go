package probes

import (
	"context"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// ShellRCProbe scans shell startup files (~/.bashrc, ~/.zshrc, ~/.profile,
// config.fish, PowerShell profiles) for hardcoded exported secrets.
type ShellRCProbe struct{}

func (ShellRCProbe) ID() string               { return "shell-rc" }
func (ShellRCProbe) Available(e env.Env) bool { return AnyPathExists(e, "shell-rc") }

func (ShellRCProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	for _, rel := range PathsFor("shell-rc", e.OS()) {
		_ = ScanTextFile(e, "shell-rc", rel, emit)
	}
	return nil
}
