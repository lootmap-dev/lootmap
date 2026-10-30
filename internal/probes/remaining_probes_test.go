package probes_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/report"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func remainingRegistry(t *testing.T) *probes.Registry {
	t.Helper()
	r := probes.NewRegistry()
	for _, p := range probes.RemainingProbes() {
		if err := r.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestRemainingProbesDepthRootsAndSkips(t *testing.T) {
	root, secrets := testutil.MaterializeHome(t, "linux-dev")

	// Add a .git/.env and a too-deep .env (depth 6 > MaxDotenvDepth=4) to verify
	// both are skipped, plus a custom root ~/custom_work/app/.env.
	gitEnv := filepath.Join(root, "code", "billing-service", ".git")
	if err := os.MkdirAll(gitEnv, 0o750); err != nil {
		t.Fatal(err)
	}
	slackTok := testutil.FakeSecret(701, "slack-token")
	if err := os.WriteFile(filepath.Join(gitEnv, ".env"), []byte("SLACK="+slackTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	deepDir := filepath.Join(root, "code", "d1", "d2", "d3", "d4", "d5")
	if err := os.MkdirAll(deepDir, 0o750); err != nil {
		t.Fatal(err)
	}
	deepTok := testutil.FakeSecret(702, "gcp-api-key")
	if err := os.WriteFile(filepath.Join(deepDir, ".env"), []byte("GCP="+deepTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}
	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}

	res, err := engine.Run(context.Background(), e, remainingRegistry(t), suite, engine.Options{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Verify node_modules, .git, .env.example, and depth>4 were skipped.
	for _, skipped := range []string{"node_modules", ".git/.env", ".env.example", "d5/.env"} {
		if strings.Contains(out, skipped) {
			t.Fatalf("expected %s to be skipped, but found in output:\n%s", skipped, out)
		}
	}
	for _, raw := range append(secrets, slackTok, deepTok) {
		if strings.Contains(out, raw) {
			t.Fatalf("output leaked raw secret")
		}
	}

	testutil.Golden(t, filepath.Join("testdata", "remaining_linux-dev.golden"), buf.Bytes())

	// Now test --roots custom directory override.
	customDir := filepath.Join(root, "custom_work", "service")
	if err := os.MkdirAll(customDir, 0o750); err != nil {
		t.Fatal(err)
	}
	customTok := testutil.FakeSecret(703, "github-pat-classic")
	if err := os.WriteFile(filepath.Join(customDir, ".env.local"), []byte("GH_TOKEN="+customTok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	resCustom, err := engine.Run(context.Background(), e, remainingRegistry(t), suite, engine.Options{
		Version:    "0.1.0",
		OnlyProbes: []string{"dotenv"},
		Roots:      []string{"~/custom_work"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resCustom.Loot) != 1 || resCustom.Loot[0].Locations[0].Path != "~/custom_work/service/.env.local" {
		t.Fatalf("expected custom root finding in ~/custom_work/service/.env.local, got %+v", resCustom.Loot)
	}
}
