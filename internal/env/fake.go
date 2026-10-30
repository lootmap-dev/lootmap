package env

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Fake is an Env for tests. Its home is a fixture directory under
// testdata/homes or a temporary directory; nothing else is accepted, so a
// test cannot accidentally point it at the real home directory.
type Fake struct {
	GOOS  string
	Vars  map[string]string
	Tools map[string]string // executable name -> fake path

	home string
	fsys fs.FS
}

// NewFake returns a Fake rooted at root. root must be an existing absolute
// directory inside a testdata/homes tree or inside os.TempDir().
func NewFake(goos, root string) (*Fake, error) {
	if root == "" {
		return nil, errors.New("env: fake root is empty")
	}
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("env: fake root %q is not absolute", root)
	}
	root = filepath.Clean(root)
	if !isFixturePath(root) && !isUnder(root, os.TempDir()) {
		return nil, fmt.Errorf("env: fake root %q is neither under testdata/homes nor the temp dir", root)
	}
	st, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("env: fake root: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("env: fake root %q is not a directory", root)
	}
	return &Fake{
		GOOS:  goos,
		Vars:  map[string]string{},
		Tools: map[string]string{},
		home:  root,
		fsys:  os.DirFS(root),
	}, nil
}

func (f *Fake) OS() string               { return f.GOOS }
func (f *Fake) Home() string             { return f.home }
func (f *Fake) Getenv(key string) string { return f.Vars[key] }
func (f *Fake) FS() fs.FS                { return readOnly{f.fsys} }

// Shorten converts host separators first: a Fake emulating darwin may run
// on a Windows host, where its home and paths use backslashes.
func (f *Fake) Shorten(p string) string {
	return ShortenPath(f.GOOS, filepath.ToSlash(f.home), filepath.ToSlash(p))
}

func (f *Fake) LookPath(name string) (string, bool) {
	p, ok := f.Tools[name]
	return p, ok
}

func isFixturePath(p string) bool {
	parts := strings.Split(filepath.ToSlash(p), "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "testdata" && parts[i+1] == "homes" && parts[i+2] != "" {
			return true
		}
	}
	return false
}

func isUnder(p, dir string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(dir), p)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
