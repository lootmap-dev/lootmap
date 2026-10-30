package cli_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

type emittingProbe struct {
	id     string
	chunks []probes.Chunk
}

func (p emittingProbe) ID() string             { return p.id }
func (p emittingProbe) Available(env.Env) bool { return true }
func (p emittingProbe) Scan(_ context.Context, _ env.Env, emit probes.Emitter) error {
	for _, c := range p.chunks {
		emit.EmitChunk(c)
	}
	return nil
}

func TestScanJSONDeduplicatesAndNeverLeaks(t *testing.T) {
	pat := testutil.FakeSecret(42, "github-pat-classic")
	openAI := testutil.FakeSecret(43, "openai-api-key")

	reg := probes.NewRegistry()
	if err := reg.Register(emittingProbe{
		id: "git",
		chunks: []probes.Chunk{
			{Probe: "git", Path: "~/.git-credentials", Line: 1, Text: "https://u:" + pat + "@github.com"},
			{Probe: "git", Path: "~/.config/gh/hosts.yml", Line: 3, Text: "oauth_token: " + pat},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(emittingProbe{
		id: "ai-transcripts",
		chunks: []probes.Chunk{
			{Probe: "ai-transcripts", Path: "~/.claude/projects/x.jsonl", Line: 19, Text: pat + " " + openAI},
		},
	}); err != nil {
		t.Fatal(err)
	}

	e := fake(t, "darwin", "mac-dev")
	out, errOut, err := run(t, e, reg, "scan", "--format", "json")
	if err != nil {
		t.Fatalf("scan --format json failed: %v (stderr: %s)", err, errOut)
	}

	for _, raw := range []string{pat, openAI, e.Home()} {
		if strings.Contains(out, raw) || strings.Contains(errOut, raw) {
			t.Fatalf("CLI output leaked %q", raw)
		}
	}

	var doc struct {
		Tool    string `json:"tool"`
		Version string `json:"version"`
		Host    struct {
			OS string `json:"os"`
		} `json:"host"`
		Score int `json:"score"`
		Loot  []struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			Locations []any  `json:"locations"`
		} `json:"loot"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal scan json: %v\n%s", err, out)
	}
	if doc.Tool != "lootmap" || doc.Host.OS != "darwin" {
		t.Fatalf("unexpected header: %+v", doc)
	}
	if len(doc.Loot) != 2 {
		t.Fatalf("expected 2 loot items, got %d: %s", len(doc.Loot), out)
	}
	for _, item := range doc.Loot {
		if item.Kind == "github-pat-classic" && len(item.Locations) != 3 {
			t.Fatalf("expected 3 locations for github-pat-classic, got %d", len(item.Locations))
		}
	}
}

func TestAllFakeSecretsNeverLeakToStdoutStderrOrJSON(t *testing.T) {
	kinds := []string{
		"github-pat-classic",
		"github-pat-fine",
		"github-oauth",
		"aws-access-key-id",
		"npm-token",
		"pypi-token",
		"openai-api-key",
		"anthropic-api-key",
		"stripe-live-key",
		"slack-token",
		"gcp-api-key",
		"private-key-pem",
		"database-url",
	}

	var secrets []string
	var chunks []probes.Chunk
	for i, k := range kinds {
		raw := testutil.FakeSecret(500+i, k)
		secrets = append(secrets, raw)
		chunks = append(chunks, probes.Chunk{
			Probe: "git",
			Path:  "~/.git-credentials",
			Line:  i + 1,
			Text:  raw,
		})
	}

	reg := probes.NewRegistry()
	if err := reg.Register(emittingProbe{id: "git", chunks: chunks}); err != nil {
		t.Fatal(err)
	}

	e := fake(t, "linux", "linux-dev")
	stdout, stderr, err := run(t, e, reg, "scan", "--format", "json")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	combined := stdout + "\n" + stderr
	for i, raw := range secrets {
		if strings.Contains(combined, raw) {
			t.Fatalf("LEAK DETECTED for %s: raw secret appeared in CLI output", kinds[i])
		}
	}
	if strings.Contains(combined, e.Home()) {
		t.Fatalf("LEAK DETECTED: absolute home directory %q appeared in CLI output", e.Home())
	}
}
