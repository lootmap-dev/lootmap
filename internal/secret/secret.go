// Package secret defines Value, the only type in lootmap that holds a raw
// secret string. Every formatting and serialization method on Value outputs a
// masked preview, making accidental leakage into logs, panics, or reports
// impossible.
package secret

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

const maskBullets = "••••••"

// Value wraps a secret so that its raw value is never exposed by String,
// GoString, Format, or MarshalJSON. Instead, only a SHA-256 fingerprint and a
// masked preview are stored and returned.
type Value struct {
	fingerprint string
	preview     string
}

// NewValue immediately hashes and masks raw, discarding the raw string so it
// does not remain reachable through Value.
func NewValue(raw string, prefixLen, suffixLen int) Value {
	sum := sha256.Sum256([]byte(raw))
	fp := "sha256:" + hex.EncodeToString(sum[:])
	return Value{
		fingerprint: fp,
		preview:     maskString(raw, prefixLen, suffixLen),
	}
}

func maskString(raw string, prefixLen, suffixLen int) string {
	if prefixLen < 0 {
		prefixLen = 0
	}
	if suffixLen < 0 {
		suffixLen = 0
	}
	if prefixLen == 0 && suffixLen == 0 {
		return maskBullets
	}
	// Require at least 4 hidden characters in the middle so short secrets are
	// never partially revealed.
	if len(raw) < prefixLen+suffixLen+4 {
		return maskBullets
	}
	return raw[:prefixLen] + maskBullets + raw[len(raw)-suffixLen:]
}

// Fingerprint returns the deterministic SHA-256 identifier ("sha256:...").
func (v Value) Fingerprint() string {
	if v.fingerprint == "" {
		return "sha256:empty"
	}
	return v.fingerprint
}

// Preview returns the masked preview (e.g. "ghp_••••••a9Xz").
func (v Value) Preview() string {
	if v.preview == "" {
		return maskBullets
	}
	return v.preview
}

// String always returns the masked preview.
func (v Value) String() string { return v.Preview() }

// GoString always returns the masked preview so %#v never leaks.
func (v Value) GoString() string { return v.Preview() }

// Format implements fmt.Formatter so every verb (%s, %v, %+v, %#v, %q) prints
// only the masked preview.
func (v Value) Format(f fmt.State, verb rune) {
	switch verb {
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", v.Preview())
	default:
		_, _ = io.WriteString(f, v.Preview())
	}
}

// MarshalJSON serializes Value as its masked preview string.
func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Preview())
}

// Match represents a single secret detected in a Chunk.
type Match struct {
	Kind  string `json:"kind"`
	Value Value  `json:"value"`
	Line  int    `json:"line"`
}
