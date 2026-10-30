package osv_test

import (
	"testing"

	"github.com/lootmap-dev/lootmap/internal/osv"
)

func FuzzParseLockfiles(f *testing.F) {
	f.Add([]byte(`{"packages":{"node_modules/pkg":{"version":"1.0.0"}}}`))
	f.Add([]byte(`packages:\n  /pkg@1.0.0:\n    resolution: {}\n`))
	f.Add([]byte("\"pkg@^1.0.0\":\n  version \"1.0.0\"\n"))
	f.Add([]byte("\x00\xff{{malformed"))

	f.Fuzz(func(_ *testing.T, data []byte) {
		_ = osv.ParsePackageLock(data)
		_ = osv.ParsePNPMLock(data)
		_ = osv.ParseYarnLock(data)
	})
}
