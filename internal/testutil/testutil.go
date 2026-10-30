// Package testutil holds helpers shared by tests. It never touches the real
// home directory: fixture homes live under testdata/homes in the repo.
package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// UpdateEnv is the environment variable that rewrites golden files:
//
//	LOOTMAP_UPDATE_GOLDEN=1 go test ./...
const UpdateEnv = "LOOTMAP_UPDATE_GOLDEN"

// RepoRoot walks up from the test's working directory to the directory
// containing go.mod.
func RepoRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above working directory")
		}
		dir = parent
	}
}

// HomePath returns the absolute path of a fixture home in testdata/homes.
func HomePath(t testing.TB, name string) string {
	t.Helper()
	return filepath.Join(RepoRoot(t), "testdata", "homes", name)
}

// NormalizeNewlines converts CRLF to LF so goldens compare equal on Windows.
func NormalizeNewlines(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// Golden compares got with the golden file at path (relative to the test's
// package directory). With LOOTMAP_UPDATE_GOLDEN=1 it rewrites the file.
func Golden(t testing.TB, path string, got []byte) {
	t.Helper()
	got = NormalizeNewlines(got)
	if os.Getenv(UpdateEnv) == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with %s=1 to create)", path, err, UpdateEnv)
	}
	want = NormalizeNewlines(want)
	if !bytes.Equal(got, want) {
		t.Fatalf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
