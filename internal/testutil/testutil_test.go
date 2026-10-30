package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func TestRepoRootHasGoMod(t *testing.T) {
	if _, err := os.Stat(filepath.Join(testutil.RepoRoot(t), "go.mod")); err != nil {
		t.Fatal(err)
	}
}

func TestHomePathFixturesExist(t *testing.T) {
	for _, name := range []string{"mac-dev", "linux-dev", "windows-dev", "clean"} {
		t.Run(name, func(t *testing.T) {
			p := testutil.HomePath(t, name)
			if st, err := os.Stat(p); err != nil || !st.IsDir() {
				t.Fatalf("fixture home %s missing: %v", name, err)
			}
		})
	}
}

func TestNormalizeNewlines(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a\r\nb\r\n", "a\nb\n"},
		{"a\nb", "a\nb"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := string(testutil.NormalizeNewlines([]byte(tc.in))); got != tc.want {
			t.Errorf("NormalizeNewlines(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
