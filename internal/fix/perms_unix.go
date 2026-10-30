//go:build !windows

package fix

import "os"

func tightenPermissionsOS(absPath string) error {
	return os.Chmod(absPath, 0o600)
}
