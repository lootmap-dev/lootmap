package loot_test

import (
	"testing"

	"github.com/lootmap-dev/lootmap/internal/loot"
)

func TestCatalogLookupAndGrades(t *testing.T) {
	cat := loot.Default()
	for _, kind := range []string{
		"github-pat-classic",
		"aws-access-key-id",
		"npm-token",
		"openai-api-key",
		"ssh-key-unencrypted",
	} {
		ent, ok := cat.Lookup(kind)
		if !ok || ent.Title == "" || ent.Impact == "" {
			t.Fatalf("Lookup(%q) = %+v, %v", kind, ent, ok)
		}
	}

	grades := []struct {
		score int
		want  string
	}{
		{100, "A"},
		{90, "A"},
		{89, "B"},
		{75, "B"},
		{60, "C"},
		{40, "D"},
		{39, "F"},
		{0, "F"},
	}
	for _, tc := range grades {
		if got := loot.Grade(tc.score); got != tc.want {
			t.Errorf("Grade(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}
