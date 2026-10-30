//go:build perf

package probes_test

import (
	"testing"
	"time"
)

// TestAITranscriptsStream2GBUnder30s verifies the SPEC.md Phase 3 acceptance
// criterion (2 GB fake transcript scanned in under 30 seconds) using an
// in-memory stream so zero bytes are written to disk.
func TestAITranscriptsStream2GBUnder30s(t *testing.T) {
	runStreamingTranscriptScan(t, 2*1024*1024*1024, 30*time.Second)
}
