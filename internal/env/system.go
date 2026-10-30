package env

import (
	"io/fs"
	"os"
	"os/exec"
	"runtime"
)

type system struct {
	home string
	fsys fs.FS
}

// System returns the Env for the real host. Only cmd/lootmap should call
// it; tests must use NewFake.
func System() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &system{home: home, fsys: os.DirFS(home)}, nil
}

func (s *system) OS() string               { return runtime.GOOS }
func (s *system) Home() string             { return s.home }
func (s *system) Getenv(key string) string { return os.Getenv(key) }
func (s *system) FS() fs.FS                { return readOnly{s.fsys} }
func (s *system) Shorten(p string) string  { return ShortenPath(runtime.GOOS, s.home, p) }

func (s *system) LookPath(name string) (string, bool) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return p, true
}

// readOnly hides any optional write methods a wrapped fs.FS might grow, so
// Env.FS stays read-only by contract.
type readOnly struct{ fs.FS }

func (r readOnly) ReadFile(name string) ([]byte, error)  { return fs.ReadFile(r.FS, name) }
func (r readOnly) Stat(name string) (fs.FileInfo, error) { return fs.Stat(r.FS, name) }
func (r readOnly) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(r.FS, name)
}
