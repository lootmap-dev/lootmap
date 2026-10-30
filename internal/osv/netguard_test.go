package osv_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/cli"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
	"github.com/lootmap-dev/lootmap/internal/testutil"
)

// TestNoNetworkInScanFixAndCheckPackages installs a tripwire on
// http.DefaultTransport and net.DefaultResolver.PreferGo/Dial so any network
// attempt during scan, fix, or check-packages immediately fails the test
// (SPEC.md network test + decision 17).
func TestNoNetworkInScanFixAndCheckPackages(t *testing.T) {
	origTransport := http.DefaultTransport
	origResolverDial := net.DefaultResolver.Dial
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
		net.DefaultResolver.Dial = origResolverDial
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("FORBIDDEN NETWORK CALL via http.DefaultTransport: %s %s", req.Method, req.URL)
		return nil, errors.New("network forbidden")
	})
	net.DefaultResolver.Dial = func(_ context.Context, network, address string) (net.Conn, error) {
		t.Fatalf("FORBIDDEN DNS/NETWORK CALL: %s %s", network, address)
		return nil, errors.New("network forbidden")
	}

	root, _ := testutil.MaterializeHome(t, "linux-dev")
	e, err := env.NewFake("linux", root)
	if err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"scan", "--format", "json"},
		{"fix", "--dry-run"},
		{"check-packages", "--format", "json"},
	} {
		var out, errOut bytes.Buffer
		cmd := cli.NewRootCmd(cli.Options{
			Env:     e,
			Probes:  probes.Default(),
			Version: "test",
			Stdout:  &out,
			Stderr:  &errOut,
		})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v failed: %v", args, err)
		}
	}
}
