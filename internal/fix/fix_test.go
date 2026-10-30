package fix_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/fix"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func toFindings(items []engine.LootItem) []probes.Finding {
	var out []probes.Finding
	for _, it := range items {
		for _, loc := range it.Locations {
			out = append(out, probes.Finding{
				Kind:        it.Kind,
				Impact:      it.Impact,
				Severity:    it.Severity,
				Fingerprint: it.Fingerprint,
				Preview:     it.Preview,
				Location:    loc,
				Fixes:       it.Fixes,
				Rotate:      it.Rotate,
			})
		}
	}
	return out
}

func TestAllFixersPlanApplyVerifyAndByteExactUndo(t *testing.T) {
	root, _ := testutil.MaterializeHome(t, "linux-dev")

	// Backdate transcript mtime by 10 minutes so live-session protection allows fixing it.
	past := time.Now().Add(-10 * time.Minute)
	transcriptPath := filepath.Join(root, ".claude", "projects", "webapp", "session-01.jsonl")
	if err := os.Chtimes(transcriptPath, past, past); err != nil {
		t.Fatal(err)
	}

	targetFiles := []string{
		".claude/projects/webapp/session-01.jsonl",
		".zsh_history",
		".gitconfig",
		".npmrc",
		".ssh/config",
		".ssh/id_ed25519",
	}

	originals := make(map[string][]byte, len(targetFiles))
	for _, rel := range targetFiles {
		b, err := os.ReadFile(filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))) // #nosec G304
		if err != nil {
			t.Fatalf("read original %s: %v", rel, err)
		}
		originals[rel] = b
	}

	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}
	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}

	resBefore, err := engine.Run(context.Background(), e, probes.Default(), suite, engine.Options{Version: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	findings := toFindings(resBefore.Loot)

	plan, err := fix.BuildPlan(e, findings, fix.Options{Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) == 0 {
		t.Fatal("expected non-empty fix plan")
	}

	bmgr := fix.NewBackupManager(root)
	manifest, err := fix.Apply(e, plan, bmgr, fix.Options{Now: time.Now})
	if err != nil {
		t.Fatalf("fix.Apply: %v", err)
	}
	if manifest.ID == "" {
		t.Fatal("expected backup manifest ID")
	}

	// Verify content changes took effect.
	afterTranscript, _ := os.ReadFile(filepath.Clean(transcriptPath)) // #nosec G304
	if !strings.Contains(string(afterTranscript), "[REDACTED:stripe-live-key]") {
		t.Fatalf("expected redacted stripe key in transcript, got:\n%s", string(afterTranscript))
	}
	afterHist, _ := os.ReadFile(filepath.Clean(filepath.Join(root, ".zsh_history"))) // #nosec G304
	if !strings.Contains(string(afterHist), "[REDACTED:openai-api-key]") {
		t.Fatalf("expected redacted openai key in .zsh_history, got:\n%s", string(afterHist))
	}
	afterGit, _ := os.ReadFile(filepath.Clean(filepath.Join(root, ".gitconfig"))) // #nosec G304
	if !strings.Contains(string(afterGit), "helper = libsecret") {
		t.Fatalf("expected helper = libsecret in .gitconfig, got:\n%s", string(afterGit))
	}
	afterNPM, _ := os.ReadFile(filepath.Clean(filepath.Join(root, ".npmrc"))) // #nosec G304
	if !strings.Contains(string(afterNPM), "ignore-scripts=true") {
		t.Fatalf("expected ignore-scripts=true in .npmrc, got:\n%s", string(afterNPM))
	}
	afterSSH, _ := os.ReadFile(filepath.Clean(filepath.Join(root, ".ssh", "config"))) // #nosec G304
	if !strings.Contains(string(afterSSH), "ForwardAgent no") {
		t.Fatalf("expected ForwardAgent no in .ssh/config, got:\n%s", string(afterSSH))
	}

	// Undo and verify every target file is byte-for-byte identical to original.
	if _, err := bmgr.Restore(manifest.ID); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	for _, rel := range targetFiles {
		got, err := os.ReadFile(filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))) // #nosec G304
		if err != nil {
			t.Fatalf("read restored %s: %v", rel, err)
		}
		if !bytes.Equal(got, originals[rel]) {
			t.Fatalf("file %s after undo is not byte-for-byte identical to original\n--- got ---\n%s\n--- want ---\n%s",
				rel, got, originals[rel])
		}
	}
}

func TestInterruptedWriteLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.json")
	original := []byte(`{"token":"original-uncorrupted-content"}`)
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}

	err := fix.AtomicWrite(target, []byte(`{"token":"half-written"}`), 0o600, true)
	if !errors.Is(err, fix.ErrSimulatedWriteCrash) {
		t.Fatalf("expected ErrSimulatedWriteCrash, got %v", err)
	}

	after, err := os.ReadFile(filepath.Clean(target)) // #nosec G304
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("interrupted write corrupted original file: got %q, want %q", after, original)
	}
}

func TestLiveSessionProtectionSkipsRecentTranscripts(t *testing.T) {
	root, _ := testutil.MaterializeHome(t, "linux-dev")
	now := time.Now()
	transcriptPath := filepath.Join(root, ".claude", "projects", "webapp", "session-01.jsonl")
	// Set mtime to 10 seconds ago (< 60s window).
	recent := now.Add(-10 * time.Second)
	if err := os.Chtimes(transcriptPath, recent, recent); err != nil {
		t.Fatal(err)
	}

	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}
	f := []probes.Finding{
		{
			Kind:     "stripe-live-key",
			Location: probes.Location{Probe: "ai-transcripts", Path: "~/.claude/projects/webapp/session-01.jsonl", Line: 1},
			Fixes:    []string{"redact-transcripts"},
		},
	}

	plan, err := fix.BuildPlan(e, f, fix.Options{Force: false, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 0 || len(plan.Skipped) != 1 {
		t.Fatalf("expected live session transcript to be skipped, got changes=%d skipped=%v", len(plan.Changes), plan.Skipped)
	}

	forcedPlan, err := fix.BuildPlan(e, f, fix.Options{Force: true, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if len(forcedPlan.Changes) != 1 {
		t.Fatalf("expected --force to include transcript change, got %d", len(forcedPlan.Changes))
	}
}

func TestBackupPrunesSnapshotsOlderThan7Days(t *testing.T) {
	home := t.TempDir()
	sample := filepath.Join(home, ".npmrc")
	if err := os.WriteFile(sample, []byte("ignore-scripts=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	baseTime := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	bmgr := fix.NewBackupManager(home)
	bmgr.Now = func() time.Time { return baseTime }

	oldSnap, err := bmgr.CreateSnapshot([]string{".npmrc"})
	if err != nil {
		t.Fatal(err)
	}

	// Advance clock by 8 days (> 7-day retention) and prune.
	bmgr.Now = func() time.Time { return baseTime.Add(8 * 24 * time.Hour) }
	pruned, err := bmgr.PruneExpired()
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 expired backup pruned, got %d", pruned)
	}
	if _, err := bmgr.Restore(oldSnap.ID); err == nil {
		t.Fatal("expected pruned snapshot restore to fail")
	}
}
