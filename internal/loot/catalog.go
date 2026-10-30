// Package loot loads the embedded catalog.yaml that maps finding kinds to
// attacker-impact descriptions, display titles, severities, fixes, and rotation
// instructions.
package loot

import (
	_ "embed"
	"sync"

	"go.yaml.in/yaml/v3"
)

//go:embed catalog.yaml
var embeddedCatalog []byte

// Entry describes one finding kind in the loot catalog.
type Entry struct {
	Kind     string   `yaml:"kind"`
	Title    string   `yaml:"title"`
	Short    string   `yaml:"short"`
	Impact   string   `yaml:"impact"`
	Severity string   `yaml:"severity"`
	Fixes    []string `yaml:"fixes"`
	Rotate   string   `yaml:"rotate"`
	Steps    []string `yaml:"steps"`
}

type catalogDoc struct {
	Entries []Entry `yaml:"entries"`
}

// Catalog holds the indexed catalog entries.
type Catalog struct {
	byKind map[string]Entry
}

var (
	defaultOnce sync.Once
	defaultCat  *Catalog
)

// Default returns the embedded loot catalog.
func Default() *Catalog {
	defaultOnce.Do(func() {
		c, err := Parse(embeddedCatalog)
		if err != nil {
			c = &Catalog{byKind: map[string]Entry{}}
		}
		defaultCat = c
	})
	return defaultCat
}

// Parse parses YAML catalog bytes into a Catalog.
func Parse(data []byte) (*Catalog, error) {
	var doc catalogDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	m := make(map[string]Entry, len(doc.Entries))
	for _, e := range doc.Entries {
		m[e.Kind] = e
	}
	return &Catalog{byKind: m}, nil
}

// Lookup returns the catalog Entry for kind, or a sensible fallback if unknown.
func (c *Catalog) Lookup(kind string) (Entry, bool) {
	if c != nil {
		if e, ok := c.byKind[kind]; ok {
			return e, true
		}
	}
	return Entry{
		Kind:     kind,
		Title:    kind,
		Short:    kind,
		Impact:   "Exposed credential or insecure configuration",
		Severity: "high",
	}, false
}

// Grade returns the letter grade for a 0-100 score (SPEC.md: A >= 90, B >= 75,
// C >= 60, D >= 40, F < 40).
func Grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}
