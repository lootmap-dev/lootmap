package probes

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// ContainersProbe scans ~/.docker/config.json (for unencrypted base64 registry
// auths without credsStore) and ~/.kube/config (for cluster tokens/keys).
type ContainersProbe struct{}

func (ContainersProbe) ID() string               { return "containers" }
func (ContainersProbe) Available(e env.Env) bool { return AnyPathExists(e, "containers") }

func (ContainersProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	for _, rel := range PathsFor("containers", e.OS()) {
		data, err := fs.ReadFile(fsys, rel)
		if err != nil {
			continue
		}
		_ = ScanTextFile(e, "containers", rel, emit)
		if strings.HasSuffix(rel, ".docker/config.json") {
			checkDockerConfig(rel, data, emit)
		}
	}
	return nil
}

type dockerConfig struct {
	CredsStore  string                       `json:"credsStore"`
	CredHelpers map[string]string            `json:"credHelpers"`
	Auths       map[string]map[string]string `json:"auths"`
}

func checkDockerConfig(rel string, data []byte, emit Emitter) {
	var cfg dockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	if cfg.CredsStore != "" {
		return
	}
	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	for registry, entry := range cfg.Auths {
		if cfg.CredHelpers != nil && cfg.CredHelpers[registry] != "" {
			continue
		}
		if auth := strings.TrimSpace(entry["auth"]); auth != "" {
			emit.EmitFinding(Finding{
				Kind:        "docker-plaintext-auth",
				Impact:      "Docker registry credentials are stored as plaintext base64 without credsStore",
				Severity:    "high",
				Fingerprint: "config:docker-plaintext-auth:" + shortPath + ":" + registry,
				Preview:     registry + " (base64 auth)",
				Location:    Location{Probe: "containers", Path: shortPath, Line: 1},
				Fixes:       []string{"file-permissions-600"},
			})
		}
	}
}
