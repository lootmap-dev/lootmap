package fix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupRetention is 7 days (SPEC.md: backups contain unmasked data and are
// automatically deleted after 7 days).
const BackupRetention = 7 * 24 * time.Hour

// BackupEntry records one backed-up file inside a backup snapshot.
type BackupEntry struct {
	RelPath string      `json:"relPath"`
	Mode    fs.FileMode `json:"mode"`
	Existed bool        `json:"existed"`
}

// BackupManifest is stored as manifest.json inside ~/.lootmap/backups/<id>/.
type BackupManifest struct {
	ID        string        `json:"id"`
	CreatedAt time.Time     `json:"createdAt"`
	Entries   []BackupEntry `json:"entries"`
}

// BackupManager manages ~/.lootmap/backups/<id>/ inside a home directory.
type BackupManager struct {
	Home string
	Now  func() time.Time
}

// NewBackupManager creates a BackupManager rooted at home.
func NewBackupManager(home string) *BackupManager {
	return &BackupManager{
		Home: home,
		Now:  time.Now,
	}
}

func (b *BackupManager) backupsRoot() string {
	return filepath.Join(b.Home, ".lootmap", "backups")
}

// PruneExpired removes backup snapshots older than BackupRetention (7 days)
// and returns the number of pruned snapshots.
func (b *BackupManager) PruneExpired() (int, error) {
	root := b.backupsRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	cutoff := b.Now().Add(-BackupRetention)
	pruned := 0
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		dir := filepath.Join(root, ent.Name())
		m, err := readManifest(dir)
		if err != nil {
			continue
		}
		if m.CreatedAt.Before(cutoff) {
			if err := os.RemoveAll(dir); err == nil {
				pruned++
			}
		}
	}
	return pruned, nil
}

// CreateSnapshot backs up the given home-relative paths into a new snapshot
// directory with 0600 permissions.
func (b *BackupManager) CreateSnapshot(relPaths []string) (BackupManifest, error) {
	_, _ = b.PruneExpired()

	now := b.Now().UTC()
	id := now.Format("20060102-150405.000000000")
	snapDir := filepath.Join(b.backupsRoot(), id)
	filesDir := filepath.Join(snapDir, "files")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return BackupManifest{}, err
	}

	seen := map[string]bool{}
	var manifestEntries []BackupEntry

	for _, rel := range relPaths {
		rel = normalizeRel(rel)
		if rel == "" || seen[rel] {
			continue
		}
		seen[rel] = true

		srcPath := filepath.Join(b.Home, filepath.FromSlash(rel))
		st, err := os.Stat(srcPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				manifestEntries = append(manifestEntries, BackupEntry{
					RelPath: rel,
					Existed: false,
				})
				continue
			}
			return BackupManifest{}, err
		}

		data, err := os.ReadFile(filepath.Clean(srcPath)) // #nosec G304 -- validated relative path inside Home
		if err != nil {
			return BackupManifest{}, err
		}
		dstPath := filepath.Join(filesDir, filepath.FromSlash(rel))
		if err := AtomicWrite(dstPath, data, 0o600, false); err != nil {
			return BackupManifest{}, err
		}
		manifestEntries = append(manifestEntries, BackupEntry{
			RelPath: rel,
			Mode:    st.Mode().Perm(),
			Existed: true,
		})
	}

	manifest := BackupManifest{
		ID:        id,
		CreatedAt: now,
		Entries:   manifestEntries,
	}
	mBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return BackupManifest{}, err
	}
	if err := AtomicWrite(filepath.Join(snapDir, "manifest.json"), mBytes, 0o600, false); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

// Restore restores the snapshot with id (or the most recent snapshot if id is
// empty) byte-for-byte.
func (b *BackupManager) Restore(id string) (BackupManifest, error) {
	root := b.backupsRoot()
	if id == "" {
		latest, err := b.latestID()
		if err != nil {
			return BackupManifest{}, err
		}
		id = latest
	}
	snapDir := filepath.Join(root, id)
	manifest, err := readManifest(snapDir)
	if err != nil {
		return BackupManifest{}, fmt.Errorf("backup %q not found: %w", id, err)
	}

	filesDir := filepath.Join(snapDir, "files")
	for _, ent := range manifest.Entries {
		target := filepath.Join(b.Home, filepath.FromSlash(ent.RelPath))
		if !ent.Existed {
			_ = os.Remove(target)
			continue
		}
		backupFile := filepath.Join(filesDir, filepath.FromSlash(ent.RelPath))
		data, err := os.ReadFile(filepath.Clean(backupFile)) // #nosec G304 -- validated relative path inside backup snapshot
		if err != nil {
			return BackupManifest{}, err
		}
		mode := ent.Mode
		if mode == 0 {
			mode = 0o600
		}
		if err := AtomicWrite(target, data, mode, false); err != nil {
			return BackupManifest{}, err
		}
	}
	return manifest, nil
}

func (b *BackupManager) latestID() (string, error) {
	entries, err := os.ReadDir(b.backupsRoot())
	if err != nil {
		return "", errors.New("no backups available")
	}
	var ids []string
	for _, ent := range entries {
		if ent.IsDir() {
			ids = append(ids, ent.Name())
		}
	}
	if len(ids) == 0 {
		return "", errors.New("no backups available")
	}
	sort.Strings(ids)
	return ids[len(ids)-1], nil
}

func readManifest(snapDir string) (BackupManifest, error) {
	data, err := os.ReadFile(filepath.Clean(filepath.Join(snapDir, "manifest.json"))) // #nosec G304 -- manifest inside backup snapshot
	if err != nil {
		return BackupManifest{}, err
	}
	var m BackupManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return BackupManifest{}, err
	}
	return m, nil
}

func normalizeRel(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	p = strings.TrimPrefix(p, "~/")
	p = strings.TrimPrefix(p, "/")
	return p
}
