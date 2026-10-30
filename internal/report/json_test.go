package report_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/report"
)

func TestWriteJSONEmptyLootArray(t *testing.T) {
	var buf bytes.Buffer
	err := report.WriteJSON(&buf, engine.Result{
		Tool:    "lootmap",
		Version: "0.1.0",
		Host:    engine.HostInfo{OS: "linux"},
		Score:   100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"loot": []`) {
		t.Fatalf("expected empty array for loot, got:\n%s", buf.String())
	}
}
