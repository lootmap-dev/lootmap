package fix

import (
	"strings"
	"testing"
)

func TestRedactDatabaseURLInsideJSONTranscript(t *testing.T) {
	rules, err := compileRedactionRegexes()
	if err != nil {
		t.Fatal(err)
	}
	pw := "Zq" + "9xR2mT7vLk4wP1sN8bQ3"
	in := `{"type":"tool_result","content":"DATABASE_URL=postgres://appuser:` + pw +
		`@db.prod.internal:5432/appdb\nNEXT=1"}`
	out, n := redactBytes([]byte(in), rules)
	if n != 1 {
		t.Fatalf("want 1 redaction, got %d: %s", n, out)
	}
	if strings.Contains(string(out), pw) {
		t.Fatalf("password survived redaction: %s", out)
	}
	if !strings.Contains(string(out), `[REDACTED:database-url]\nNEXT=1"}`) {
		t.Fatalf("JSON escape or trailing content damaged: %s", out)
	}
}

func TestRedactLeavesPasswordlessURLAlone(t *testing.T) {
	rules, err := compileRedactionRegexes()
	if err != nil {
		t.Fatal(err)
	}
	in := "DATABASE_URL=postgres://localhost:5432/app"
	out, n := redactBytes([]byte(in), rules)
	if n != 0 || string(out) != in {
		t.Fatalf("unexpected redaction: %d %s", n, out)
	}
}
