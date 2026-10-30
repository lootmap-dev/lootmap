package probes

import (
	"context"
	"io/fs"
	"path"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// SSHProbe scans ~/.ssh for unencrypted private keys, ForwardAgent yes in
// ~/.ssh/config, and overly permissive file modes on Unix.
type SSHProbe struct{}

func (SSHProbe) ID() string               { return "ssh" }
func (SSHProbe) Available(e env.Env) bool { return AnyPathExists(e, "ssh") }

func (SSHProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	entries, err := fs.ReadDir(fsys, ".ssh")
	if err != nil {
		return nil
	}

	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if strings.HasSuffix(name, ".pub") || name == "known_hosts" || name == "known_hosts.old" || name == "authorized_keys" {
			continue
		}
		rel := path.Join(".ssh", name)
		shortPath := "~/" + rel
		data, err := fs.ReadFile(fsys, rel)
		if err != nil {
			continue
		}
		text := string(data)

		if name == "config" {
			scanSSHConfig(shortPath, text, emit)
			continue
		}

		if strings.Contains(text, "PRIVATE KEY-----") {
			// Emit the private key chunk for secret detection.
			_ = ScanTextFile(e, "ssh", rel, emit)

			// Check if the private key is unencrypted (no ENCRYPTED header).
			if !strings.Contains(text, "ENCRYPTED") {
				emit.EmitFinding(Finding{
					Kind:        "ssh-key-unencrypted",
					Impact:      "Passphraseless SSH private key can be used immediately if stolen",
					Severity:    "high",
					Fingerprint: "config:ssh-key-unencrypted:" + shortPath,
					Preview:     name + " (no passphrase)",
					Location:    Location{Probe: "ssh", Path: shortPath, Line: 1},
					Fixes:       []string{"ssh-permissions"},
				})
			}

			if e.OS() != "windows" {
				if info, err := ent.Info(); err == nil && info.Mode().Perm()&0o077 != 0 {
					emit.EmitFinding(Finding{
						Kind:        "ssh-key-permissions",
						Impact:      "SSH private key has overly permissive file mode",
						Severity:    "high",
						Fingerprint: "config:ssh-key-permissions:" + shortPath,
						Preview:     info.Mode().Perm().String(),
						Location:    Location{Probe: "ssh", Path: shortPath, Line: 1},
						Fixes:       []string{"ssh-permissions"},
					})
				}
			}
		}
	}
	return nil
}

func scanSSHConfig(shortPath, content string, emit Emitter) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for idx, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(strings.ReplaceAll(line, "=", " "))
		if len(fields) >= 2 && strings.EqualFold(fields[0], "ForwardAgent") && strings.EqualFold(fields[1], "yes") {
			emit.EmitFinding(Finding{
				Kind:        "ssh-forward-agent",
				Impact:      "SSH agent forwarding exposes local keys to compromised remote hosts",
				Severity:    "high",
				Fingerprint: "config:ssh-forward-agent:" + shortPath,
				Preview:     "ForwardAgent yes",
				Location:    Location{Probe: "ssh", Path: shortPath, Line: idx + 1},
				Fixes:       []string{"ssh-disable-forward-agent"},
			})
		}
	}
}
