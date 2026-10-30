package probes_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
)

type stubProbe struct {
	id        string
	available bool
}

func (s stubProbe) ID() string                                          { return s.id }
func (s stubProbe) Available(env.Env) bool                              { return s.available }
func (s stubProbe) Scan(context.Context, env.Env, probes.Emitter) error { return nil }

func TestRegistry(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantIDs []string
		wantErr bool
	}{
		{"empty", nil, []string{}, false},
		{"single", []string{"git"}, []string{"git"}, false},
		{"sorted by id", []string{"ssh", "cloud", "git"}, []string{"cloud", "git", "ssh"}, false},
		{"duplicate rejected", []string{"git", "git"}, []string{"git"}, true},
		{"empty id rejected", []string{""}, []string{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := probes.NewRegistry()
			var gotErr error
			for _, id := range tc.ids {
				if err := r.Register(stubProbe{id: id}); err != nil {
					gotErr = err
				}
			}
			if (gotErr != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", gotErr, tc.wantErr)
			}
			got := []string{}
			for _, p := range r.All() {
				got = append(got, p.ID())
			}
			if !reflect.DeepEqual(got, tc.wantIDs) {
				t.Fatalf("All() = %v, want %v", got, tc.wantIDs)
			}
		})
	}
}

func TestRegistryRejectsNil(t *testing.T) {
	if err := probes.NewRegistry().Register(nil); err == nil {
		t.Fatal("Register(nil) should fail")
	}
}

func TestDefaultRegistryContainsProbes(t *testing.T) {
	if n := len(probes.Default().All()); n < 6 {
		t.Fatalf("Default() should contain at least the 6 classic probes, got %d", n)
	}
}
