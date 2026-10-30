// Package probes holds the probe registry, the scan input/output types
// (Chunk, Finding, Emitter), and from Phase 2 on, the probes themselves.
package probes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// Chunk is a piece of raw text read by a probe to be scanned by detectors.
//
// Chunk.Text carries raw scan input, so String, GoString, Format, and
// MarshalJSON always mask Text to prevent accidental logging or serialization.
type Chunk struct {
	Probe string
	Path  string // shortened with ~
	Line  int
	Text  string
}

func (c Chunk) maskedSummary() string {
	return fmt.Sprintf("Chunk{Probe:%s Path:%s Line:%d Text:[REDACTED]}", c.Probe, c.Path, c.Line)
}

// String masks Text so a Chunk can never leak raw input to logs.
func (c Chunk) String() string { return c.maskedSummary() }

// GoString masks Text so %#v never leaks raw input.
func (c Chunk) GoString() string { return c.maskedSummary() }

// Format implements fmt.Formatter and always masks Text.
func (c Chunk) Format(f fmt.State, verb rune) {
	if verb == 'q' {
		_, _ = fmt.Fprintf(f, "%q", c.maskedSummary())
		return
	}
	_, _ = fmt.Fprint(f, c.maskedSummary())
}

// MarshalJSON omits raw Text and replaces it with "[REDACTED]".
func (c Chunk) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Probe string `json:"probe"`
		Path  string `json:"path"`
		Line  int    `json:"line"`
		Text  string `json:"text"`
	}{
		Probe: c.Probe,
		Path:  c.Path,
		Line:  c.Line,
		Text:  "[REDACTED]",
	})
}

// Location identifies where a finding was observed.
type Location struct {
	Probe string `json:"probe"`
	Path  string `json:"path"` // shortened with ~
	Line  int    `json:"line,omitempty"`
}

// Finding represents a single discovered issue at one Location. Probes emit
// config/permission findings directly via EmitFinding; secret findings are
// produced by running Detectors over emitted Chunks.
type Finding struct {
	Kind        string   `json:"kind"`
	Impact      string   `json:"impact"`
	Severity    string   `json:"severity"` // critical | high | medium | low
	Fingerprint string   `json:"fingerprint"`
	Preview     string   `json:"preview"`
	Location    Location `json:"location"`
	Fixes       []string `json:"fixes,omitempty"`
	Rotate      string   `json:"rotate,omitempty"`
}

// Emitter receives Chunks (for secret detection) and direct Findings (for
// non-secret configuration/permission issues) from a running Probe.
type Emitter interface {
	EmitChunk(c Chunk)
	EmitFinding(f Finding)
}

// Probe knows how to read one place on disk.
type Probe interface {
	ID() string               // e.g. "ai-transcripts"
	Available(e env.Env) bool // is the tool installed / does the path exist
	Scan(ctx context.Context, e env.Env, emit Emitter) error
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

var defaultRegistry = func() *Registry {
	r := NewRegistry()
	for _, p := range ClassicProbes() {
		_ = r.Register(p)
	}
	for _, p := range AIProbes() {
		_ = r.Register(p)
	}
	return r
}()

// ClassicProbes returns the 6 Phase 2 probes (cloud, git, npm, ssh,
// shell-history, shell-rc).
func ClassicProbes() []Probe {
	return []Probe{
		CloudProbe{},
		GitProbe{},
		NPMProbe{},
		SSHProbe{},
		ShellHistoryProbe{},
		ShellRCProbe{},
	}
}

// AIProbes returns the Phase 3 AI probes (ai-transcripts, ai-config).
func AIProbes() []Probe {
	return []Probe{
		AITranscriptsProbe{},
		AIConfigProbe{},
	}
}

// Default returns the registry used by the lootmap binary.
func Default() *Registry { return defaultRegistry }
