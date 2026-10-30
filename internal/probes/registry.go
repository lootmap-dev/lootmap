// Package probes holds the probe registry and, from Phase 2 on, the probes
// themselves.
package probes

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// Probe knows how to read one place on disk.
//
// Phase 0 defines only identity and availability; Scan (with the Emitter
// interface) is added in Phase 1.
type Probe interface {
	ID() string               // e.g. "ai-transcripts"
	Available(e env.Env) bool // is the tool installed / does the path exist
}

// Registry is a set of probes keyed by ID.
type Registry struct {
	mu     sync.RWMutex
	probes map[string]Probe
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{probes: map[string]Probe{}}
}

// Register adds p. IDs must be non-empty and unique.
func (r *Registry) Register(p Probe) error {
	if p == nil {
		return errors.New("probes: nil probe")
	}
	id := p.ID()
	if id == "" {
		return errors.New("probes: empty probe id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.probes[id]; dup {
		return fmt.Errorf("probes: duplicate probe id %q", id)
	}
	r.probes[id] = p
	return nil
}

// All returns the registered probes sorted by ID.
func (r *Registry) All() []Probe {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Probe, 0, len(r.probes))
	for _, p := range r.probes {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

var defaultRegistry = NewRegistry()

// Default returns the registry used by the lootmap binary.
func Default() *Registry { return defaultRegistry }
