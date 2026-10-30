package probes

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/osv"
)

// PackagesProbe scans lockfiles (package-lock.json, pnpm-lock.yaml, yarn.lock)
// in development folders and npm caches against the offline OSV MAL- database.
// It runs in both 'lootmap scan' and 'lootmap check-packages' (decision 12).
type PackagesProbe struct{}

func (PackagesProbe) ID() string { return "packages" }

func (PackagesProbe) Available(e env.Env) bool {
	if AnyPathExists(e, "packages") {
		return true
	}
	return DotenvProbe{}.Available(e)
}

func (PackagesProbe) Scan(ctx context.Context, e env.Env, emit Emitter) error {
	db, err := osv.Default(e)
	if err != nil {
		return err
	}

	fsys := e.FS()
	roots := (DotenvProbe{}).effectiveRoots(e)
	for _, root := range roots {
		st, err := fs.Stat(fsys, root)
		if err != nil || !st.IsDir() {
			continue
		}
		walkLockfiles(ctx, e, db, root, 0, MaxDotenvDepth, emit)
	}

	for _, cachePath := range PathsFor("packages", e.OS()) {
		st, err := fs.Stat(fsys, cachePath)
		if err != nil || !st.IsDir() {
			continue
		}
		walkLockfiles(ctx, e, db, cachePath, 0, 4, emit)
	}
	return nil
}

func walkLockfiles(ctx context.Context, e env.Env, db *osv.DB, dir string, depth, maxDepth int, emit Emitter) {
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
				walkLockfiles(ctx, e, db, full, depth+1, maxDepth, emit)
			}
			continue
		}
		switch name {
		case "package-lock.json", "npm-shrinkwrap.json":
			scanLockfile(e, db, full, osv.ParsePackageLock, emit)
		case "pnpm-lock.yaml":
			scanLockfile(e, db, full, osv.ParsePNPMLock, emit)
		case "yarn.lock":
			scanLockfile(e, db, full, osv.ParseYarnLock, emit)
		}
	}
}

func scanLockfile(e env.Env, db *osv.DB, rel string, parser func([]byte) []osv.PackageRef, emit Emitter) {
	data, err := fs.ReadFile(e.FS(), rel)
	if err != nil {
		return
	}
	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	for _, ref := range parser(data) {
		adv, ok := db.Lookup(ref.Name, ref.Version)
		if !ok {
			continue
		}
		emit.EmitFinding(Finding{
			Kind:        "malicious-package",
			Impact:      fmt.Sprintf("%s: %s (%s@%s)", adv.ID, adv.Summary, ref.Name, ref.Version),
			Severity:    "critical",
			Fingerprint: fmt.Sprintf("osv:%s:%s@%s", adv.ID, ref.Name, ref.Version),
			Preview:     fmt.Sprintf("%s@%s (%s)", ref.Name, ref.Version, adv.ID),
			Location:    Location{Probe: "packages", Path: shortPath, Line: ref.Line},
		})
	}
}
