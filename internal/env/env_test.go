package env_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func TestShortenPath(t *testing.T) {
	tests := []struct {
		name string
		goos string
		home string
		path string
		want string
	}{
		{"unix home itself", "linux", "/home/dev", "/home/dev", "~"},
		{"unix file under home", "linux", "/home/dev", "/home/dev/.aws/credentials", "~/.aws/credentials"},
		{"unix trailing slash on home", "darwin", "/Users/dev/", "/Users/dev/.npmrc", "~/.npmrc"},
		{"unix sibling prefix is not home", "linux", "/home/dev", "/home/developer/.npmrc", "/home/developer/.npmrc"},
		{"unix outside home", "linux", "/home/dev", "/etc/passwd", "/etc/passwd"},
		{"unix case sensitive", "linux", "/home/dev", "/home/DEV/.npmrc", "/home/DEV/.npmrc"},
		{"unix unclean path", "linux", "/home/dev", "/home/dev/a/../.npmrc", "~/.npmrc"},
		{"windows backslashes", "windows", `C:\Users\dev`, `C:\Users\dev\.aws\credentials`, "~/.aws/credentials"},
		{"windows case insensitive", "windows", `C:\Users\Dev`, `c:\users\dev\AppData\x.json`, "~/AppData/x.json"},
		{"windows mixed separators", "windows", `C:\Users\dev`, `C:/Users/dev/.ssh/id_ed25519`, "~/.ssh/id_ed25519"},
		{"windows home itself", "windows", `C:\Users\dev`, `C:\Users\dev\`, "~"},
		{"windows outside home", "windows", `C:\Users\dev`, `D:\code\.env`, "D:/code/.env"},
		{"windows sibling prefix", "windows", `C:\Users\dev`, `C:\Users\dev2\.npmrc`, "C:/Users/dev2/.npmrc"},
		{"empty home never matches", "linux", "", "/home/dev/.npmrc", "/home/dev/.npmrc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := env.ShortenPath(tc.goos, tc.home, tc.path)
			if got != tc.want {
				t.Fatalf("ShortenPath(%q, %q, %q) = %q, want %q", tc.goos, tc.home, tc.path, got, tc.want)
			}
		})
	}
}

func TestFake(t *testing.T) {
	tests := []struct {
		name string
		home string
		goos string
	}{
		{"mac", "mac-dev", "darwin"},
		{"linux", "linux-dev", "linux"},
		{"windows", "windows-dev", "windows"},
		{"clean", "clean", "linux"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := testutil.HomePath(t, tc.home)
			f, err := env.NewFake(tc.goos, root)
			if err != nil {
				t.Fatalf("NewFake: %v", err)
			}
			f.Vars["SHELL"] = "/bin/zsh"
			f.Tools["git"] = "/usr/bin/git"

			var e env.Env = f
			if e.OS() != tc.goos {
				t.Errorf("OS() = %q, want %q", e.OS(), tc.goos)
			}
			if e.Home() != root {
				t.Errorf("Home() = %q, want %q", e.Home(), root)
			}
			if got := e.Getenv("SHELL"); got != "/bin/zsh" {
				t.Errorf("Getenv(SHELL) = %q", got)
			}
			if got := e.Getenv("MISSING"); got != "" {
				t.Errorf("Getenv(MISSING) = %q, want empty", got)
			}
			if p, ok := e.LookPath("git"); !ok || p != "/usr/bin/git" {
				t.Errorf("LookPath(git) = %q, %v", p, ok)
			}
			if _, ok := e.LookPath("docker"); ok {
				t.Errorf("LookPath(docker) should be false")
			}
			if _, err := fs.Stat(e.FS(), ".gitkeep"); err != nil {
				t.Errorf("FS should expose fake home contents: %v", err)
			}
			if got := e.Shorten(filepath.Join(root, ".aws", "credentials")); got != "~/.aws/credentials" {
				t.Errorf("Shorten = %q", got)
			}
		})
	}
}

func TestFakeFSIsReadOnly(t *testing.T) {
	f, err := env.NewFake("linux", testutil.HomePath(t, "clean"))
	if err != nil {
		t.Fatal(err)
	}
	// The read-only contract: FS must not offer any write capability.
	if _, ok := f.FS().(interface {
		WriteFile(string, []byte, fs.FileMode) error
	}); ok {
		t.Fatal("FS must be read-only")
	}
}

func TestNewFakeRejectsUnsafeRoots(t *testing.T) {
	tests := []struct {
		name string
		root string
	}{
		{"empty", ""},
		{"relative", "testdata/homes/clean"},
		{"not a fixture or temp dir", filepath.Join(testutil.RepoRoot(t), "internal")},
		{"nonexistent fixture", filepath.Join(testutil.RepoRoot(t), "testdata", "homes", "does-not-exist")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := env.NewFake("linux", tc.root); err == nil {
				t.Fatalf("NewFake(%q) should fail", tc.root)
			}
		})
	}
}

func TestNewFakeAcceptsTempDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := env.NewFake("linux", dir); err != nil {
		t.Fatalf("NewFake(t.TempDir()) = %v", err)
	}
}
