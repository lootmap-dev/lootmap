package report_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/report"
)

func TestWriteHTMLIsOfflineAndComplete(t *testing.T) {
	var buf bytes.Buffer
	res := sampleResult()
	if err := report.WriteHTML(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, `content="default-src 'none'`) {
		t.Fatal("HTML report must ship a Content-Security-Policy that blocks all network loads")
	}
	// Nothing may be loaded from elsewhere: no external src/href except plain
	// revoke links (<a href>), which the browser only follows on click.
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)<(script|img|iframe|link)[^>]+(src|href)=["']https?:`),
		regexp.MustCompile(`(?i)@import|url\(\s*["']?https?:`),
	} {
		if m := re.FindString(out); m != "" {
			t.Fatalf("HTML report references a remote resource: %q", m)
		}
	}
	for _, it := range res.Loot {
		if !strings.Contains(out, it.Preview) {
			t.Errorf("preview %q missing from report", it.Preview)
		}
		for _, l := range it.Locations {
			if !strings.Contains(out, l.Path) {
				t.Errorf("location %q missing from report", l.Path)
			}
		}
	}
}

func TestWriteHTMLCleanMachine(t *testing.T) {
	var buf bytes.Buffer
	res := engine.Result{Tool: "lootmap", Version: "0.1.0", Host: engine.HostInfo{OS: "linux"}, Score: 100}
	if err := report.WriteHTML(&buf, res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Nothing worth stealing was found.") {
		t.Fatal("clean report should say nothing was found")
	}
}

func TestWriteHTMLEscapesFindingText(t *testing.T) {
	var buf bytes.Buffer
	res := sampleResult()
	res.Loot[0].Locations[0].Path = `~/<script>alert(1)</script>`
	if err := report.WriteHTML(&buf, res); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "<script>alert(1)</script>") {
		t.Fatal("file paths must be HTML-escaped")
	}
}
