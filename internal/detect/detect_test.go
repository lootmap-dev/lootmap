package detect_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

func TestChunkNeverLeaksRawText(t *testing.T) {
	raw := testutil.FakeSecret(10, "github-pat-classic")
	c := probes.Chunk{
		Probe: "git",
		Path:  "~/.git-credentials",
		Line:  1,
		Text:  "https://user:" + raw + "@github.com",
	}

	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	outs := []string{
		c.String(),
		c.GoString(),
		fmt.Sprintf("%s", c),
		fmt.Sprintf("%v", c),
		fmt.Sprintf("%+v", c),
		fmt.Sprintf("%#v", c),
		string(b),
	}
	for i, out := range outs {
		if strings.Contains(out, raw) {
			t.Fatalf("Chunk representation [%d] leaked raw text: %s", i, out)
		}
	}
}

func TestDefaultDetectorsMatchFakeSecrets(t *testing.T) {
	suite, err := detect.Default()
	if err != nil {
		t.Fatalf("detect.Default: %v", err)
	}

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

	for i, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			raw := testutil.FakeSecret(100+i, kind)
			line := fmt.Sprintf("export SECRET_%d=%q", i, raw)
			if kind == "private-key-pem" || kind == "database-url" {
				line = raw
			}
			chunk := probes.Chunk{
				Probe: "test",
				Path:  "~/.env",
				Line:  i + 1,
				Text:  line,
			}
			matches := suite.FindAll(chunk)
			if len(matches) == 0 {
				t.Fatalf("expected match for %s, got none", kind)
			}
			found := false
			for _, m := range matches {
				if m.Kind == kind {
					found = true
				}
				if strings.Contains(m.Value.Preview(), raw) {
					t.Fatalf("match preview leaked raw secret %s", kind)
				}
			}
			if !found {
				t.Fatalf("matches %v did not include kind %q", matches, kind)
			}
		})
	}
}

func TestFalsePositivesIgnored(t *testing.T) {
	suite, err := detect.Default()
	if err != nil {
		t.Fatalf("detect.Default: %v", err)
	}

	raw := testutil.FakeSecret(55, "github-pat-classic")

	cases := []struct {
		name string
		path string
		text string
	}{
		{"dotenv example file", "~/.env.example", "GITHUB_TOKEN=" + raw},
		{"dotenv sample file", "~/project/.env.sample", "GITHUB_TOKEN=" + raw},
		{"dotenv template file", "~/project/.env.template", "GITHUB_TOKEN=" + raw},
		{"aws official example key", "~/.aws/credentials", "aws_access_key_id = AKIAIOSFODNN7EXAMPLE"},
		{"env var expansion brace", "~/.zshrc", "export GITHUB_TOKEN=${GITHUB_TOKEN}"},
		{"env var expansion dollar", "~/.zshrc", "export NPM_TOKEN=$NPM_TOKEN"},
		{"placeholder changeme", "~/.env", "DATABASE_URL=postgres://user:changeme@localhost:5432/db"},
		{"placeholder your_api_key", "~/.env", "OPENAI_API_KEY=your_api_key_here"},
		{"placeholder xxx", "~/.env", "GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		{"placeholder todo", "~/.env", "STRIPE_KEY=sk_live_TODO_REPLACE_ME_0000000000"},
		{"redacted marker", "~/.claude/projects/a.jsonl", "[REDACTED:github-pat-classic]"},
		{"db url without password", "~/.env", "DATABASE_URL=postgres://localhost:5432/appdb"},
		{"db url with empty password", "~/.env", "DATABASE_URL=postgres://user:@localhost:5432/appdb"},
		{"db url with env var password", "~/.env", "DATABASE_URL=postgres://user:${DB_PASS}@localhost:5432/appdb"},
		{"data uri base64 image", "~/notes.md", "![img](data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUA====)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches := suite.FindAll(probes.Chunk{
				Probe: "test",
				Path:  tc.path,
				Line:  1,
				Text:  tc.text,
			})
			if len(matches) != 0 {
				t.Fatalf("expected 0 matches for %s, got %+v", tc.name, matches)
			}
		})
	}
}
