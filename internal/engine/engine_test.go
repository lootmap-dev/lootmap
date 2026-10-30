package engine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/report"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

type fakeProbe struct {
	id       string
	chunks   []probes.Chunk
	findings []probes.Finding
}

func (f fakeProbe) ID() string             { return f.id }
func (f fakeProbe) Available(env.Env) bool { return true }
func (f fakeProbe) Scan(_ context.Context, _ env.Env, emit probes.Emitter) error {
	for _, c := range f.chunks {
		emit.EmitChunk(c)
	}
	for _, fn := range f.findings {
		emit.EmitFinding(fn)
	}
	return nil
}

func TestDeduplicationAndLeakFreeJSON(t *testing.T) {
	pat := testutil.FakeSecret(1, "github-pat-classic")
	npmTok := testutil.FakeSecret(2, "npm-token")
	awsID := testutil.FakeSecret(3, "aws-access-key-id")

	reg := probes.NewRegistry()
	if err := reg.Register(fakeProbe{
		id: "git",
		chunks: []probes.Chunk{
			{Probe: "git", Path: "~/.git-credentials", Line: 1, Text: "https://u:" + pat + "@github.com"},
			{Probe: "git", Path: "~/.config/gh/hosts.yml", Line: 4, Text: "oauth_token: " + pat},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(fakeProbe{
		id: "ai-transcripts",
		chunks: []probes.Chunk{
			{Probe: "ai-transcripts", Path: "~/.claude/projects/app/session.jsonl", Line: 812, Text: `{"text":"token is ` + pat + `"}`},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(fakeProbe{
		id: "npm",
		chunks: []probes.Chunk{
			{Probe: "npm", Path: "~/.npmrc", Line: 2, Text: "//registry.npmjs.org/:_authToken=" + npmTok},
		},
		findings: []probes.Finding{
			{
				Kind:        "npm-ignore-scripts-disabled",
				Impact:      "npm install scripts can execute arbitrary code on install",
				Severity:    "high",
				Fingerprint: "config:npm-ignore-scripts-disabled",
				Preview:     "ignore-scripts=false",
				Location:    probes.Location{Probe: "npm", Path: "~/.npmrc", Line: 1},
				Fixes:       []string{"npm-ignore-scripts"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(fakeProbe{
		id: "cloud",
		chunks: []probes.Chunk{
			{Probe: "cloud", Path: "~/.aws/credentials", Line: 2, Text: "aws_access_key_id = " + awsID},
		},
	}); err != nil {
		t.Fatal(err)
	}

	suite, err := detect.Default()
	if err != nil {
		t.Fatal(err)
	}
	e, err := env.NewFake("linux", testutil.HomePath(t, "linux-dev"))
	if err != nil {
		t.Fatal(err)
	}

	res, err := engine.Run(context.Background(), e, reg, suite, engine.Options{Version: "0.1.0"})
	if err != nil {
		t.Fatalf("engine.Run: %v", err)
	}

	// 3 distinct secrets + 1 config finding = 4 loot items.
	if len(res.Loot) != 4 {
		t.Fatalf("len(Loot) = %d, want 4: %+v", len(res.Loot), res.Loot)
	}

	var patLoot *engine.LootItem
	for i := range res.Loot {
		if res.Loot[i].Kind == "github-pat-classic" {
			patLoot = &res.Loot[i]
		}
		if !strings.HasPrefix(res.Loot[i].ID, "L-") || len(res.Loot[i].ID) < len("L-")+8 {
			t.Errorf("unexpected ID format %q", res.Loot[i].ID)
		}
	}
	if patLoot == nil {
		t.Fatal("github-pat-classic loot item not found")
	}
	if len(patLoot.Locations) != 3 {
		t.Fatalf("expected 3 deduplicated locations for github PAT, got %d: %+v", len(patLoot.Locations), patLoot.Locations)
	}

	var buf bytes.Buffer
	if err := report.WriteJSON(&buf, res); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	jsonOut := buf.String()

	// Leak check: none of the raw secrets or absolute home path may appear in JSON.
	for _, secretVal := range []string{pat, npmTok, awsID, e.Home()} {
		if strings.Contains(jsonOut, secretVal) {
			t.Fatalf("JSON output leaked sensitive value %q:\n%s", secretVal, jsonOut)
		}
	}
	if strings.Contains(jsonOut, "hostnameHash") {
		t.Fatalf("JSON output must not contain hostnameHash:\n%s", jsonOut)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}
