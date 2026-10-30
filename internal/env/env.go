// Package env abstracts the operating system and the user's home directory.
//
// Every probe, detector and command reaches the file system only through an
// Env. Production code uses System(); tests use Fake, which is rooted in a
// fixture under testdata/homes or in a t.TempDir(), so tests never read or
// modify the real home directory.
//
// Env is read-only by design. Writing (used by fixers) goes through a
// separate layer added in Phase 6.
package env

import (
	"io/fs"
	"path"
	"strings"
)

// Env is the only gateway to the host environment.
type Env interface {
	// OS returns the operating system in runtime.GOOS form
	// ("darwin", "linux", "windows").
	OS() string
	// Home returns the absolute home directory path.
	Home() string
	// Getenv returns the value of an environment variable, or "".
	Getenv(key string) string
	// FS returns a read-only file system rooted at Home. Paths are
	// slash-separated and relative to the home directory.
	FS() fs.FS
	// LookPath reports whether an executable is on PATH and where.
	// It never runs the executable.
	LookPath(name string) (string, bool)
	// Shorten replaces the home directory prefix of p with "~" and uses
	// forward slashes, so reports never contain the user name.
	Shorten(p string) string
}

// ShortenPath is the pure implementation behind Env.Shorten. On Windows the
// comparison is case-insensitive and both separators are accepted; the
// result always uses forward slashes.
func ShortenPath(goos, home, p string) string {
	if goos == "windows" {
		home = strings.ReplaceAll(home, `\`, "/")
		p = strings.ReplaceAll(p, `\`, "/")
	}
	p = cleanSlash(p)
	if home == "" {
		return p
	}
	home = cleanSlash(home)

	equal := func(a, b string) bool { return a == b }
	if goos == "windows" {
		equal = strings.EqualFold
	}
	if equal(p, home) {
		return "~"
	}
	prefix := strings.TrimSuffix(home, "/") + "/"
	if len(p) > len(prefix) && equal(p[:len(prefix)], prefix) {
		return "~/" + p[len(prefix):]
	}
	return p
}

// cleanSlash cleans a slash-separated path while preserving a Windows drive
// prefix such as "C:".
func cleanSlash(p string) string {
	if p == "" {
		return ""
	}
	return path.Clean(p)
}
