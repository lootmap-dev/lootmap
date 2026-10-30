package testutil

import (
	"fmt"
	"math/rand"
	"strings"
)

const (
	alnum      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	upperAlnum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowerHex   = "0123456789abcdef"
)

func seededChars(seed int64, salt string, alphabet string, n int) string {
	h := seed*1000003 + 97
	for i := 0; i < len(salt); i++ {
		h = h*31 + int64(salt[i])
	}
	r := rand.New(rand.NewSource(h))
	var b strings.Builder
	b.Grow(n)
	for i := 0; i < n; i++ {
		b.WriteByte(alphabet[r.Intn(len(alphabet))])
	}
	return b.String()
}

// FakeSecret generates a realistic-format fake secret at runtime from a
// deterministic seed so no real-format secret is ever committed to the repo.
func FakeSecret(seed int, kind string) string {
	s := int64(seed)
	switch kind {
	case "github-pat-classic":
		return "ghp_" + seededChars(s, kind, alnum, 36)
	case "github-pat-fine":
		return "github_pat_" + seededChars(s, kind, alnum, 22) + "_" + seededChars(s+1, kind, alnum, 59)
	case "github-oauth":
		return "gho_" + seededChars(s, kind, alnum, 36)
	case "aws-access-key-id":
		return "AKIA" + seededChars(s, kind, upperAlnum, 16)
	case "aws-secret-access-key":
		return seededChars(s, kind, alnum+"+/", 40)
	case "npm-token":
		return "npm_" + seededChars(s, kind, alnum, 36)
	case "pypi-token":
		return "pypi-AgEIcHlwaS5vcmcC" + seededChars(s, kind, alnum+"-_", 50)
	case "openai-api-key":
		return "sk" + "-proj-" + seededChars(s, kind, alnum, 48)
	case "anthropic-api-key":
		return "sk" + "-ant-api03-" + seededChars(s, kind, alnum+"-_", 48)
	case "stripe-live-key":
		return "sk" + "_live_" + seededChars(s, kind, alnum, 24)
	case "slack-token":
		return "xoxb-" + seededChars(s, kind, "0123456789", 11) + "-" + seededChars(s+1, kind, "0123456789", 11) + "-" + seededChars(s+2, kind, alnum, 24)
	case "gcp-api-key":
		return "AIza" + seededChars(s, kind, alnum+"-_", 35)
	case "private-key-pem":
		body := seededChars(s, kind, alnum+"+/", 64)
		return "-----BEGIN OPENSSH PRIVATE KEY-----\n" + body + "\n-----END OPENSSH PRIVATE KEY-----"
	case "database-url":
		pw := seededChars(s, kind, alnum, 20)
		return fmt.Sprintf("postgres://appuser:%s@db.prod.internal:5432/appdb", pw)
	default:
		return "tok_" + seededChars(s, kind, lowerHex, 32)
	}
}
