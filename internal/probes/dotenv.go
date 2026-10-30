package probes

import (
	"context"
	"io/fs"
	"path"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// MaxDotenvDepth is the maximum directory depth below each development root
// searched by DotenvProbe.
const MaxDotenvDepth = 4

// DotenvProbe scans development folders for scattered .env* files while
// skipping node_modules, .git, vendor, and .env.example templates.
type DotenvProbe struct {
	// Roots overrides the default paths.yaml roots when non-empty (relative to
	// home, or ~ prefixed).
	Roots    []string
	MaxDepth int
}

func (DotenvProbe) ID() string { return "dotenv" }

func (d DotenvProbe) Available(e env.Env) bool {
	for _, r := range d.effectiveRoots(e) {
		if _, err := fs.Stat(e.FS(), r); err == nil {
			return true
		}
	}
	return false
}

func (d DotenvProbe) effectiveRoots(e env.Env) []string {
	if len(d.Roots) > 0 {
		out := make([]string, 0, len(d.Roots))
		for _, r := range d.Roots {
			s := e.Shorten(strings.TrimSpace(r))
			s = strings.TrimPrefix(s, "~/")
			s = strings.TrimPrefix(s, "/")
			if s != "" && s != "~" {
				out = append(out, s)
			}
		}
		return out
	}
	if custom := e.Getenv("LOOTMAP_ROOTS"); custom != "" {
		var out []string
		for _, r := range strings.Split(custom, ",") {
			s := e.Shorten(strings.TrimSpace(r))
			s = strings.TrimPrefix(s, "~/")
			s = strings.TrimPrefix(s, "/")
			if s != "" && s != "~" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return PathsFor("dotenv", e.OS())
}

func (d DotenvProbe) Scan(ctx context.Context, e env.Env, emit Emitter) error {
	maxDepth := d.MaxDepth
	if maxDepth <= 0 {
		maxDepth = MaxDotenvDepth
	}
	fsys := e.FS()
	for _, root := range d.effectiveRoots(e) {
		st, err := fs.Stat(fsys, root)
		if err != nil || !st.IsDir() {
			continue
		}
		walkDotenvDir(ctx, e, root, 0, maxDepth, emit)
	}
	return nil
}

func walkDotenvDir(ctx context.Context, e env.Env, dir string, depth, maxDepth int, emit Emitter) {
	if ctx.Err() != nil || depth > maxDepth {
		return
	}
	entries, err := fs.ReadDir(e.FS(), dir)
	if err != nil {
		return
	}
	for _, ent := range entries {
		name := ent.Name()
		full := path.Join(dir, name)
		if ent.IsDir() {
			if shouldSkipDir(name) {
				continue
			}
			if depth+1 <= maxDepth {
				walkDotenvDir(ctx, e, full, depth+1, maxDepth, emit)
			}
			continue
		}
		if isDotenvCandidate(name) {
			_ = ScanTextFile(e, "dotenv", full, emit)
		}
	}
}

func shouldSkipDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", "node_modules", "vendor", "dist", "build", "target", ".next", ".venv", "venv", "__pycache__":
		return true
	}
	return false
}

func isDotenvCandidate(name string) bool {
	low := strings.ToLower(name)
	if low != ".env" && !strings.HasPrefix(low, ".env.") {
		return false
	}
	switch low {
	case ".env.example", ".env.sample", ".env.template", ".env.dist", ".env.schema":
		return false
	}
	return true
}
