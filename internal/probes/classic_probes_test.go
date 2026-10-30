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

func classicRegistry(t *testing.T) *probes.Registry {
	t.Helper()
	r := probes.NewRegistry()
	for _, p := range probes.ClassicProbes() {
		if err := r.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestCleanHomeProducesZeroFindings(t *testing.T) {
	root, secrets := testutil.MaterializeHome(t, "clean")
	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}
	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}
	res, err := engine.Run(context.Background(), e, classicRegistry(t), suite, engine.Options{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Loot) != 0 {
		t.Fatalf("clean home must produce 0 findings, got %d: %+v", len(res.Loot), res.Loot)
	}
	if res.Score != 100 {
		t.Fatalf("clean home score = %d, want 100", res.Score)
	}
	_ = secrets
}

func TestClassicProbesGolden(t *testing.T) {
	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		goos   string
		home   string
		golden string
	}{
		{"linux-dev", "linux", "linux-dev", "classic_linux-dev.golden"},
		{"mac-dev", "darwin", "mac-dev", "classic_mac-dev.golden"},
		{"windows-dev", "windows", "windows-dev", "classic_windows-dev.golden"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, secrets := testutil.MaterializeHome(t, tc.home)
			// Fixtures are materialized with mode 0600. Make the SSH key
			// group/world-readable so the permission check fires the same way
			// on every host OS (Windows always reports 0666).
			if tc.goos != "windows" {
				key := filepath.Join(root, ".ssh", "id_ed25519")
				if _, statErr := os.Stat(key); statErr == nil {
					if err := os.Chmod(key, 0o644); err != nil { // #nosec G302 -- intentional loose permissions for test
						t.Fatal(err)
					}
				}
			}
			e, err := env.NewFake(tc.goos, root)
			if err != nil {
				t.Fatal(err)
			}
			res, err := engine.Run(context.Background(), e, classicRegistry(t), suite, engine.Options{Version: "0.1.0"})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Loot) == 0 {
				t.Fatalf("expected findings for %s, got 0", tc.name)
			}

			var buf bytes.Buffer
			if err := report.WriteJSON(&buf, res); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			for _, raw := range secrets {
				if strings.Contains(out, raw) {
					t.Fatalf("golden JSON leaked raw secret in %s", tc.name)
				}
			}
			if strings.Contains(out, root) {
				t.Fatalf("golden JSON leaked temp root path %q", root)
			}
			testutil.Golden(t, filepath.Join("testdata", tc.golden), buf.Bytes())
		})
	}
}
