// Command gen expands docs/demo-src/home.txtar into a demo home directory,
// replacing {{PLACEHOLDER}} tokens with the same deterministic fake secrets
// the test suite uses. It exists only to record the README demo.
//
// Usage: go run ./docs/demo-src/gen docs/demo-src/home.txtar /tmp/lootmap-demo/home
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: gen <home.txtar> <output-dir>")
		os.Exit(2)
	}
	data, err := os.ReadFile(filepath.Clean(os.Args[1])) // #nosec G304,G703 -- CLI tool reading user-specified txtar
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dst := os.Args[2]
	tokens := testutil.DefaultTokens()

	var name string
	var body strings.Builder
	flush := func() {
		if name == "" {
			return
		}
		content := body.String()
		for k, v := range tokens {
			content = strings.ReplaceAll(content, k, v)
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil { // #nosec G301,G703
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Clean(target), []byte(content), 0o600); err != nil { // #nosec G304,G703
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	for _, line := range strings.SplitAfter(string(data), "\n") {
		trimmed := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(trimmed, "-- ") && strings.HasSuffix(trimmed, " --") && len(trimmed) > 6 {
			flush()
			name = strings.TrimSuffix(strings.TrimPrefix(trimmed, "-- "), " --")
			body.Reset()
			continue
		}
		if name != "" {
			body.WriteString(line)
		}
	}
	flush()
}
