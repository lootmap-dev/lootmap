//go:build windows

package fix

import "os"

// tightenPermissionsOS applies owner-only mode (0600); BackupManager preserves
// the original mode and contents for undo.
func tightenPermissionsOS(absPath string) error {
	return os.Chmod(absPath, 0o600)
}
