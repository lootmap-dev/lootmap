package fix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrSimulatedWriteCrash is injected by tests to simulate an interrupted write
// before the atomic rename step.
var ErrSimulatedWriteCrash = errors.New("simulated write crash before atomic rename")

// AtomicWrite writes data to a sibling temporary file in the same directory,
// syncs it to disk, sets permissions to 0600 (or mode), and atomically renames
// it over targetPath. If failBeforeRename is true, the temp file is removed and
// targetPath remains untouched.
func AtomicWrite(targetPath string, data []byte, mode os.FileMode, failBeforeRename bool) error {
	if mode == 0 {
		mode = 0o600
	}
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".lootmap-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("chmod temp: %w", err)
	}

	if failBeforeRename {
		_ = os.Remove(tmpName)
		return ErrSimulatedWriteCrash
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
