package probes_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/report"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func aiRegistry(t *testing.T) *probes.Registry {
	t.Helper()
	r := probes.NewRegistry()
	for _, p := range probes.AIProbes() {
		if err := r.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestAIProbesGoldenAndSQLite(t *testing.T) {
	root, secrets := testutil.MaterializeHome(t, "linux-dev")

	// Also synthesize a small binary Cursor state.vscdb file containing a fake
	// GitHub OAuth token surrounded by binary SQLite header/padding bytes.
	oauth := testutil.DefaultTokens()["{{GITHUB_OAUTH_1}}"]
	vscdbDir := filepath.Join(root, ".config", "Cursor", "User", "globalStorage")
	if err := os.MkdirAll(vscdbDir, 0o750); err != nil {
		t.Fatal(err)
	}
	var dbPayload bytes.Buffer
	dbPayload.WriteString("SQLite format 3\x00\x10\x00\x01\x01\x00@  ")
	dbPayload.Write([]byte{0x00, 0x01, 0x02, 0x00})
	dbPayload.WriteString(`{"chat":"token ` + oauth + ` in cursor"}`)
	dbPayload.Write([]byte{0x00, 0xff, 0x00})
	if err := os.WriteFile(filepath.Join(vscdbDir, "state.vscdb"), dbPayload.Bytes(), 0o600); err != nil {
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

	res, err := engine.Run(context.Background(), e, aiRegistry(t), suite, engine.Options{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Loot) < 5 {
		t.Fatalf("expected at least 5 AI findings, got %d: %+v", len(res.Loot), res.Loot)
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, raw := range secrets {
		if strings.Contains(out, raw) {
			t.Fatalf("AI probe output leaked raw secret")
		}
	}
	testutil.Golden(t, filepath.Join("testdata", "ai_linux-dev.golden"), buf.Bytes())
}

// streamingFS wraps an in-memory generated stream so tests can verify multi-MB
// and multi-GB transcript scanning with zero disk usage.
type streamingFS struct {
	relPath   string
	totalSize int64
	lineBytes []byte
	secretAt  int64
	secretRow []byte
}

func (s *streamingFS) Open(name string) (fs.File, error) {
	if name == ".claude/projects" {
		return &memDir{name: "projects", entry: &memDirEntry{name: "huge.jsonl", size: s.totalSize}}, nil
	}
	if name == s.relPath {
		return &streamFile{
			total:     s.totalSize,
			line:      s.lineBytes,
			secretPos: s.secretAt,
			secretRow: s.secretRow,
		}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *streamingFS) Stat(name string) (fs.FileInfo, error) {
	if name == ".claude/projects" {
		return &memDirEntry{name: "projects", dir: true}, nil
	}
	if name == s.relPath {
		return &memDirEntry{name: "huge.jsonl", size: s.totalSize}, nil
	}
	return nil, fs.ErrNotExist
}

func (s *streamingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == ".claude/projects" {
		return []fs.DirEntry{&memDirEntry{name: "huge.jsonl", size: s.totalSize}}, nil
	}
	return nil, fs.ErrNotExist
}

type memDir struct {
	name  string
	entry *memDirEntry
	done  bool
}

func (d *memDir) Stat() (fs.FileInfo, error) { return &memDirEntry{name: d.name, dir: true}, nil }
func (d *memDir) Read([]byte) (int, error)   { return 0, io.EOF }
func (d *memDir) Close() error               { return nil }
func (d *memDir) ReadDir(int) ([]fs.DirEntry, error) {
	if d.done {
		return nil, nil
	}
	d.done = true
	return []fs.DirEntry{d.entry}, nil
}

type memDirEntry struct {
	name string
	size int64
	dir  bool
}

func (e *memDirEntry) Name() string { return e.name }
func (e *memDirEntry) Size() int64  { return e.size }
func (e *memDirEntry) Mode() fs.FileMode {
	if e.dir {
		return fs.ModeDir | 0o755
	}
	return 0o600
}
func (e *memDirEntry) ModTime() time.Time         { return time.Time{} }
func (e *memDirEntry) IsDir() bool                { return e.dir }
func (e *memDirEntry) Sys() any                   { return nil }
func (e *memDirEntry) Type() fs.FileMode          { return e.Mode().Type() }
func (e *memDirEntry) Info() (fs.FileInfo, error) { return e, nil }

type streamFile struct {
	pos       int64
	total     int64
	line      []byte
	secretPos int64
	secretRow []byte
	injected  bool
}

func (f *streamFile) Stat() (fs.FileInfo, error) {
	return &memDirEntry{name: "huge.jsonl", size: f.total}, nil
}
func (f *streamFile) Close() error { return nil }

func (f *streamFile) Read(p []byte) (int, error) {
	if f.pos >= f.total {
		return 0, io.EOF
	}
	if !f.injected && f.pos >= f.secretPos && len(p) >= len(f.secretRow) {
		f.injected = true
		n := copy(p, f.secretRow)
		f.pos += int64(n)
		return n, nil
	}
	written := 0
	for written+len(f.line) <= len(p) && f.pos < f.total {
		n := copy(p[written:], f.line)
		written += n
		f.pos += int64(n)
	}
	if written == 0 {
		rem := f.total - f.pos
		if int64(len(p)) < rem {
			rem = int64(len(p))
		}
		n := copy(p, f.line[:rem])
		f.pos += int64(n)
		return n, nil
	}
	return written, nil
}

type customFSEnv struct {
	env.Env
	fsys fs.FS
}

func (c customFSEnv) FS() fs.FS { return c.fsys }

func runStreamingTranscriptScan(t *testing.T, sizeBytes int64, maxDur time.Duration) {
	t.Helper()
	baseEnv, err := env.NewFake("linux", testutil.HomePath(t, "clean"))
	if err != nil {
		t.Fatal(err)
	}
	pat := testutil.FakeSecret(900, "github-pat-classic")
	benignLine := []byte(`{"type":"assistant","text":"normal benign conversation line without any credentials at all 0123456789abcdef"}` + "\n")
	secretLine := []byte(`{"type":"user","text":"leaked token ` + pat + ` inside huge transcript"}` + "\n")

	sfs := &streamingFS{
		relPath:   ".claude/projects/huge.jsonl",
		totalSize: sizeBytes,
		lineBytes: benignLine,
		secretAt:  sizeBytes / 2,
		secretRow: secretLine,
	}

	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	res, err := engine.Run(context.Background(), customFSEnv{Env: baseEnv, fsys: sfs}, aiRegistry(t), suite, engine.Options{Version: "0.1.0"})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > maxDur {
		t.Fatalf("scan of %d bytes took %v, want <%v", sizeBytes, elapsed, maxDur)
	}
	if len(res.Loot) != 1 || res.Loot[0].Kind != "github-pat-classic" {
		t.Fatalf("expected 1 github-pat-classic finding in streamed transcript, got %+v", res.Loot)
	}
}

func TestAITranscriptsStream64MBInMemory(t *testing.T) {
	runStreamingTranscriptScan(t, 64*1024*1024, 10*time.Second)
}
