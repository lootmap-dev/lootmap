// Command lootmap shows what malware would take from this computer.
package main

import (
	"os"

	"github.com/lootmap-dev/lootmap/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Execute(version, os.Args[1:]))
}
