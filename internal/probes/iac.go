package probes

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/secret"
)

// IaCProbe scans ~/.terraform.d/credentials.tfrc.json and ~/.pulumi/credentials.json.
type IaCProbe struct{}

func (IaCProbe) ID() string               { return "iac" }
func (IaCProbe) Available(e env.Env) bool { return AnyPathExists(e, "iac") }

func (IaCProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	for _, rel := range PathsFor("iac", e.OS()) {
		data, err := fs.ReadFile(fsys, rel)
		if err != nil {
			continue
		}
		_ = ScanTextFile(e, "iac", rel, emit)
		if strings.HasSuffix(rel, "credentials.tfrc.json") {
			checkTerraformCredentials(rel, data, emit)
		}
	}
	return nil
}

type tfrcDoc struct {
	Credentials map[string]struct {
		Token string `json:"token"`
	} `json:"credentials"`
}

func checkTerraformCredentials(rel string, data []byte, emit Emitter) {
	var doc tfrcDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return
	}
	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	for host, cred := range doc.Credentials {
		tok := strings.TrimSpace(cred.Token)
		if len(tok) < 12 {
			continue
		}
		val := secret.NewValue(tok, 6, 4)
		emit.EmitFinding(Finding{
			Kind:        "terraform-cloud-token",
			Impact:      "Read Terraform state (often containing cloud/DB secrets) and trigger runs on " + host,
			Severity:    "medium",
			Fingerprint: val.Fingerprint(),
			Preview:     val.Preview(),
			Location:    Location{Probe: "iac", Path: shortPath, Line: 1},
			Rotate:      "https://app.terraform.io/app/settings/tokens",
		})
	}
}
