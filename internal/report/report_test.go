package report_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/report"
)

func sampleResult() engine.Result {
	return engine.Result{
		Tool:    "lootmap",
		Version: "0.1.0",
		Host:    engine.HostInfo{OS: "darwin"},
		Score:   20,
		Loot: []engine.LootItem{
			{
				ID:          "L-a1b2c3d4",
				Kind:        "aws-access-key-id",
				Impact:      "Long-lived access to your AWS account",
				Severity:    "critical",
				Fingerprint: "sha256:a1b2c3d4",
				Preview:     "AKIA••••••7890",
				Locations: []probes.Location{
					{Probe: "cloud", Path: "~/.aws/credentials", Line: 2},
				},
				Fixes:  []string{"file-permissions-600"},
				Rotate: "https://console.aws.amazon.com/iam/home#/security_credentials",
			},
			{
				ID:          "L-e5f60718",
				Kind:        "github-pat-classic",
				Impact:      "Write access to your GitHub repositories and packages",
				Severity:    "critical",
				Fingerprint: "sha256:e5f60718",
				Preview:     "ghp_••••••a9Xz",
				Locations: []probes.Location{
					{Probe: "git", Path: "~/.git-credentials", Line: 1},
					{Probe: "git", Path: "~/.config/gh/hosts.yml", Line: 3},
					{Probe: "ai-transcripts", Path: "~/.claude/projects/very-long-project-directory-name/session-transcript.jsonl", Line: 812},
				},
				Fixes:  []string{"redact-transcripts", "git-credential-helper"},
				Rotate: "https://github.com/settings/tokens",
			},
		},
	}
}

func TestWritePrettyFitsIn80Columns(t *testing.T) {
	var buf bytes.Buffer
	if err := report.WritePretty(&buf, sampleResult(), report.PrettyOptions{NoColor: true, ProbesCount: 12}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "IF MALWARE RAN ON THIS COMPUTER IT WOULD TAKE:") {
		t.Fatalf("missing headline in pretty output:\n%s", out)
	}
	if !strings.Contains(out, "Score: 20/100 (F)") {
		t.Fatalf("missing score line in pretty output:\n%s", out)
	}
	for idx, line := range strings.Split(out, "\n") {
		w := ansi.StringWidth(line)
		if w > 80 {
			t.Errorf("line %d exceeds 80 columns (width=%d): %q", idx+1, w, line)
		}
	}
}

func TestWriteMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := report.WriteMarkdown(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"# lootmap report (darwin)", "**Score:** 20/100 (F)", "`L-a1b2c3d4`", "`ghp_••••••a9Xz`"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown missing %q:\n%s", want, out)
		}
	}
}
