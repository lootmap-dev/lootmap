package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/cli"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

type stubProbe struct {
	id        string
	available bool
}

func (s stubProbe) ID() string             { return s.id }
func (s stubProbe) Available(env.Env) bool { return s.available }

func run(t *testing.T, e env.Env, reg *probes.Registry, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	if reg == nil {
		reg = probes.NewRegistry()
	}
	cmd := cli.NewRootCmd(cli.Options{
		Env:     e,
		Probes:  reg,
		Version: "test",
		Stdout:  &out,
		Stderr:  &errOut,
	})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func fake(t *testing.T, goos, home string) *env.Fake {
	t.Helper()
	f, err := env.NewFake(goos, testutil.HomePath(t, home))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestHelp(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		golden string
	}{
		{"root --help", []string{"--help"}, "help.golden"},
		{"no args shows help", []string{}, "help.golden"},
		{"doctor --help", []string{"doctor", "--help"}, "doctor_help.golden"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, _, err := run(t, fake(t, "linux", "clean"), nil, tc.args...)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			testutil.Golden(t, filepath.Join("testdata", tc.golden), []byte(out))
		})
	}
}

func TestDoctor(t *testing.T) {
	withProbes := probes.NewRegistry()
	for _, p := range []stubProbe{{"git", true}, {"cloud", false}} {
		if err := withProbes.Register(p); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name   string
		goos   string
		home   string
		reg    *probes.Registry
		golden string
	}{
		{"mac-dev", "darwin", "mac-dev", nil, "doctor_mac-dev.golden"},
		{"linux-dev", "linux", "linux-dev", nil, "doctor_linux-dev.golden"},
		{"windows-dev", "windows", "windows-dev", nil, "doctor_windows-dev.golden"},
		{"clean", "linux", "clean", nil, "doctor_clean.golden"},
		{"with stub probes", "linux", "linux-dev", withProbes, "doctor_with_probes.golden"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := fake(t, tc.goos, tc.home)
			out, _, err := run(t, e, tc.reg, "doctor")
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if strings.Contains(out, e.Home()) {
				t.Fatalf("doctor must not print the absolute home path")
			}
			testutil.Golden(t, filepath.Join("testdata", tc.golden), []byte(out))
		})
	}
}

func TestVersionFlag(t *testing.T) {
	out, _, err := run(t, fake(t, "linux", "clean"), nil, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "lootmap version test" {
		t.Fatalf("--version = %q", out)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	if _, _, err := run(t, fake(t, "linux", "clean"), nil, "definitely-not-a-command"); err == nil {
		t.Fatal("unknown command should return an error")
	}
}

func TestNoCompletionCommand(t *testing.T) {
	if _, _, err := run(t, fake(t, "linux", "clean"), nil, "completion", "bash"); err == nil {
		t.Fatal("default completion command should be disabled")
	}
}
