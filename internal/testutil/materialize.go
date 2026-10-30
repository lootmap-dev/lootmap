package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// DefaultTokens returns the deterministic map of template placeholders
// ({{GITHUB_PAT_1}}, etc.) to runtime-generated fake secrets. Because fixture
// files only contain {{PLACEHOLDER}} tokens, no real-format secret is ever
// committed to git.
func DefaultTokens() map[string]string {
	return map[string]string{
		"{{GITHUB_PAT_1}}":    FakeSecret(101, "github-pat-classic"),
		"{{GITHUB_PAT_2}}":    FakeSecret(102, "github-pat-fine"),
		"{{GITHUB_OAUTH_1}}":  FakeSecret(103, "github-oauth"),
		"{{AWS_KEY_ID_1}}":    FakeSecret(104, "aws-access-key-id"),
		"{{AWS_SECRET_1}}":    FakeSecret(105, "aws-secret-access-key"),
		"{{NPM_TOKEN_1}}":     FakeSecret(106, "npm-token"),
		"{{PYPI_TOKEN_1}}":    FakeSecret(107, "pypi-token"),
		"{{OPENAI_KEY_1}}":    FakeSecret(108, "openai-api-key"),
		"{{ANTHROPIC_KEY_1}}": FakeSecret(109, "anthropic-api-key"),
		"{{STRIPE_KEY_1}}":    FakeSecret(110, "stripe-live-key"),
		"{{SLACK_TOKEN_1}}":   FakeSecret(111, "slack-token"),
		"{{GCP_KEY_1}}":       FakeSecret(112, "gcp-api-key"),
		"{{PRIVATE_KEY_1}}":   FakeSecret(113, "private-key-pem"),
		"{{DB_URL_1}}":        FakeSecret(114, "database-url"),
	}
}

// MaterializeHome copies testdata/homes/<name> into a fresh t.TempDir(),
// replacing any {{TOKEN}} placeholders with runtime-generated fake secrets.
// It returns the temporary root path and the slice of all generated raw secrets
// so tests can assert that none of them leak.
func MaterializeHome(t testing.TB, name string) (string, []string) {
	t.Helper()
	srcRoot := HomePath(t, name)
	dstRoot := t.TempDir()
	tokens := DefaultTokens()

	var secrets []string
	for _, v := range tokens {
		secrets = append(secrets, v)
	}

	err := filepath.WalkDir(srcRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		content := string(data)
		for k, v := range tokens {
			content = strings.ReplaceAll(content, k, v)
		}
		return os.WriteFile(target, []byte(content), 0o600)
	})
	if err != nil {
		t.Fatalf("MaterializeHome(%s): %v", name, err)
	}
	return dstRoot, secrets
}
