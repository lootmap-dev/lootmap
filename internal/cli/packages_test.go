package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func TestCheckPackagesCatchesMaliciousLockfile(t *testing.T) {
	root, _ := testutil.MaterializeHome(t, "clean")
	projDir := filepath.Join(root, "code", "compromised-app")
	if err := os.MkdirAll(projDir, 0o750); err != nil {
		t.Fatal(err)
	}
	lockContent := `{
  "name": "compromised-app",
  "lockfileVersion": 3,
  "packages": {
    "node_modules/@ctrl/tinycolor": {
      "version": "4.1.1"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(projDir, "package-lock.json"), []byte(lockContent), 0o600); err != nil {
		t.Fatal(err)
	}

	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := run(t, e, probes.Default(), "check-packages", "--format", "json")
	if err != nil {
		t.Fatalf("check-packages failed: %v", err)
	}
	if !strings.Contains(out, "MAL-2025-4912") || !strings.Contains(out, "@ctrl/tinycolor@4.1.1") {
		t.Fatalf("expected MAL-2025-4912 in check-packages output, got:\n%s", out)
	}

	// Verify scan also runs the shared packages probe and catches it.
	scanOut, _, err := run(t, e, probes.Default(), "scan", "--probe", "packages", "--format", "json")
	if err != nil {
		t.Fatalf("scan --probe packages failed: %v", err)
	}
	if !strings.Contains(scanOut, "MAL-2025-4912") {
		t.Fatalf("expected scan to catch MAL-2025-4912 via packages probe, got:\n%s", scanOut)
	}
}
