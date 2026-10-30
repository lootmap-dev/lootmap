// Package archtest enforces SPEC.md architecture rules on the source tree:
//   - network packages may only be imported by internal/osv;
//   - os/exec may only be imported by internal/env (LookPath);
//   - os.UserHomeDir may only be called from internal/env, so every other
//     package reaches the home directory through the Env interface.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/testutil"
)

type rule struct {
	imp     string
	allowed []string // slash-separated directory prefixes relative to repo root
}

var importRules = []rule{
	{"net", []string{"internal/osv"}},
	{"net/http", []string{"internal/osv"}},
	{"net/http/httptrace", []string{"internal/osv"}},
	{"net/http/httputil", []string{"internal/osv"}},
	{"net/rpc", nil},
	{"net/smtp", nil},
	{"crypto/tls", []string{"internal/osv"}},
	{"os/exec", []string{"internal/env"}},
}

var homeDirAllowed = []string{"internal/env"}

func allowedIn(dir string, prefixes []string) bool {
	for _, p := range prefixes {
		if dir == p || strings.HasPrefix(dir, p+"/") {
			return true
		}
	}
	return false
}

type goFile struct {
	rel  string // slash path relative to repo root
	dir  string
	file *ast.File
}

func parseTree(t *testing.T) []goFile {
	t.Helper()
	root := testutil.RepoRoot(t)
	var files []goFile
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "testdata", "dist", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files = append(files, goFile{rel: rel, dir: filepath.ToSlash(filepath.Dir(rel)), file: f})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no Go files found; archtest is misconfigured")
	}
	return files
}

func TestImportRules(t *testing.T) {
	for _, gf := range parseTree(t) {
		for _, is := range gf.file.Imports {
			path, _ := strconv.Unquote(is.Path.Value)
			for _, r := range importRules {
				if path == r.imp && !allowedIn(gf.dir, r.allowed) {
					t.Errorf("%s imports %q; allowed only in %v", gf.rel, path, r.allowed)
				}
			}
		}
	}
}

func TestUserHomeDirOnlyInEnv(t *testing.T) {
	for _, gf := range parseTree(t) {
		if allowedIn(gf.dir, homeDirAllowed) {
			continue
		}
		ast.Inspect(gf.file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "os" && sel.Sel.Name == "UserHomeDir" {
				t.Errorf("%s calls os.UserHomeDir; use env.Env instead", gf.rel)
			}
			return true
		})
	}
}

func TestRuleHelpers(t *testing.T) {
	tests := []struct {
		dir      string
		prefixes []string
		want     bool
	}{
		{"internal/osv", []string{"internal/osv"}, true},
		{"internal/osv/update", []string{"internal/osv"}, true},
		{"internal/osvx", []string{"internal/osv"}, false},
		{"internal/cli", []string{"internal/osv"}, false},
		{"internal/cli", nil, false},
	}
	for _, tc := range tests {
		if got := allowedIn(tc.dir, tc.prefixes); got != tc.want {
			t.Errorf("allowedIn(%q, %v) = %v, want %v", tc.dir, tc.prefixes, got, tc.want)
		}
	}
}

var usesLineRe = regexp.MustCompile(`^\s*-?\s*uses:\s*([^\s#]+)`)
var sha40Re = regexp.MustCompile(`^[0-9a-f]{40}$`)

func TestGitHubActionsPinnedToFullSHA(t *testing.T) {
	wfDir := filepath.Join(testutil.RepoRoot(t), ".github", "workflows")
	entries, err := os.ReadDir(wfDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if ent.IsDir() || (!strings.HasSuffix(ent.Name(), ".yml") && !strings.HasSuffix(ent.Name(), ".yaml")) {
			continue
		}
		p := filepath.Join(wfDir, ent.Name())
		data, err := os.ReadFile(filepath.Clean(p)) // #nosec G304
		if err != nil {
			t.Fatal(err)
		}
		for idx, line := range strings.Split(string(data), "\n") {
			m := usesLineRe.FindStringSubmatch(line)
			if len(m) != 2 {
				continue
			}
			ref := m[1]
			// slsa-github-generator requires a tag ref (@vX.Y.Z) for provenance verification.
			if strings.HasPrefix(ref, "slsa-framework/slsa-github-generator/") {
				if !strings.HasSuffix(ref, "@v2.1.0") {
					t.Errorf("%s:%d slsa-github-generator must use @v2.1.0 tag, got %q", ent.Name(), idx+1, ref)
				}
				continue
			}
			at := strings.LastIndexByte(ref, '@')
			if at < 0 || !sha40Re.MatchString(ref[at+1:]) {
				t.Errorf("%s:%d action %q is not pinned to a 40-char commit SHA", ent.Name(), idx+1, ref)
			}
		}
	}
}
