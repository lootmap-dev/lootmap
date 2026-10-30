package probes

import (
	"context"
	"io/fs"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// CloudProbe scans ~/.aws/credentials, ~/.aws/config, ~/.config/gcloud/, and ~/.azure/.
type CloudProbe struct{}

func (CloudProbe) ID() string               { return "cloud" }
func (CloudProbe) Available(e env.Env) bool { return AnyPathExists(e, "cloud") }

func (CloudProbe) Scan(_ context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	for _, rel := range PathsFor("cloud", e.OS()) {
		st, err := fs.Stat(fsys, rel)
		if err != nil {
			continue
		}
		if st.IsDir() {
			_ = fs.WalkDir(fsys, rel, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				return ScanTextFile(e, "cloud", p, emit)
			})
			continue
		}
		_ = ScanTextFile(e, "cloud", rel, emit)
	}
	return nil
}
