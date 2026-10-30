// Package engine runs available probes in parallel, passes emitted Chunks
// through the Detector suite, deduplicates findings by fingerprint into Loot
// items, and computes the 0-100 security score.
package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	"github.com/lootmap-dev/lootmap/internal/detect"
	"github.com/lootmap-dev/lootmap/internal/env"
	"github.com/lootmap-dev/lootmap/internal/probes"
)

// Options configures a scan run.
type Options struct {
	Version    string
	OnlyProbes []string // optional filter of probe IDs
	Roots      []string // optional custom development roots for dotenv/packages
}

// HostInfo describes the scanned host without exposing the hostname or username.
type HostInfo struct {
	OS string `json:"os"`
}

// LootItem is a deduplicated secret or configuration issue grouped by
// fingerprint across one or more Locations.
type LootItem struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Impact      string            `json:"impact"`
	Severity    string            `json:"severity"`
	Fingerprint string            `json:"fingerprint"`
	Preview     string            `json:"preview"`
	Locations   []probes.Location `json:"locations"`
	Fixes       []string          `json:"fixes,omitempty"`
	Rotate      string            `json:"rotate,omitempty"`
}

// Result is the complete output of an engine scan.
type Result struct {
	Tool    string     `json:"tool"`
	Version string     `json:"version"`
	Host    HostInfo   `json:"host"`
	Score   int        `json:"score"`
	Loot    []LootItem `json:"loot"`
}

type collector struct {
	mu       sync.Mutex
	e        env.Env
	suite    *detect.Suite
	findings []probes.Finding
}

func (c *collector) EmitChunk(ch probes.Chunk) {
	ch.Path = c.e.Shorten(ch.Path)
	matches := c.suite.FindAll(ch)
	if len(matches) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range matches {
		meta, ok := c.suite.Meta(m.Kind)
		if !ok {
			meta = detect.RuleMeta{
				ID:       m.Kind,
				Severity: "high",
				Impact:   "Exposed credential",
			}
		}
		c.findings = append(c.findings, probes.Finding{
			Kind:        m.Kind,
			Impact:      meta.Impact,
			Severity:    meta.Severity,
			Fingerprint: m.Value.Fingerprint(),
			Preview:     m.Value.Preview(),
			Location: probes.Location{
				Probe: ch.Probe,
				Path:  ch.Path,
				Line:  m.Line,
			},
			Fixes:  meta.Fixes,
			Rotate: meta.Rotate,
		})
	}
}

func (c *collector) EmitFinding(f probes.Finding) {
	f.Location.Path = c.e.Shorten(f.Location.Path)
	if f.Fingerprint == "" {
		sum := sha256.Sum256([]byte(f.Kind + ":" + f.Location.Path + ":" + f.Preview))
		f.Fingerprint = "sha256:" + hex.EncodeToString(sum[:])
	}
	c.mu.Lock()
	c.findings = append(c.findings, f)
	c.mu.Unlock()
}

// Run executes all selected and available probes concurrently, deduplicates
// findings by fingerprint, assigns collision-free IDs, and calculates the score.
func Run(ctx context.Context, e env.Env, reg *probes.Registry, suite *detect.Suite, opts Options) (Result, error) {
	if len(opts.Roots) > 0 {
		e = rootsEnv{Env: e, roots: strings.Join(opts.Roots, ",")}
	}
	selected := selectProbes(reg.All(), e, opts.OnlyProbes)

	col := &collector{e: e, suite: suite}
	var wg sync.WaitGroup
	errCh := make(chan error, len(selected))

	for _, pr := range selected {
		wg.Add(1)
		go func(p probes.Probe) {
			defer wg.Done()
			if err := p.Scan(ctx, e, col); err != nil {
				errCh <- err
			}
		}(pr)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return Result{}, err
		}
	}

	loot := deduplicate(col.findings)
	ver := opts.Version
	if ver == "" {
		ver = "dev"
	}
	return Result{
		Tool:    "lootmap",
		Version: ver,
		Host:    HostInfo{OS: e.OS()},
		Score:   CalculateScore(loot),
		Loot:    loot,
	}, nil
}

type rootsEnv struct {
	env.Env
	roots string
}

func (r rootsEnv) Getenv(key string) string {
	if key == "LOOTMAP_ROOTS" && r.roots != "" {
		return r.roots
	}
	return r.Env.Getenv(key)
}

func selectProbes(all []probes.Probe, e env.Env, only []string) []probes.Probe {
	allow := map[string]bool{}
	for _, id := range only {
		id = strings.TrimSpace(id)
		if id != "" {
			allow[id] = true
		}
	}
	var out []probes.Probe
	for _, p := range all {
		if len(allow) > 0 && !allow[p.ID()] {
			continue
		}
		if p.Available(e) {
			out = append(out, p)
		}
	}
	return out
}

func deduplicate(findings []probes.Finding) []LootItem {
	byFP := make(map[string]*LootItem)
	var order []string

	for _, f := range findings {
		fp := f.Fingerprint
		item, ok := byFP[fp]
		if !ok {
			cpFixes := append([]string(nil), f.Fixes...)
			item = &LootItem{
				Kind:        f.Kind,
				Impact:      f.Impact,
				Severity:    f.Severity,
				Fingerprint: fp,
				Preview:     f.Preview,
				Fixes:       cpFixes,
				Rotate:      f.Rotate,
			}
			byFP[fp] = item
			order = append(order, fp)
		}
		if !hasLocation(item.Locations, f.Location) {
			item.Locations = append(item.Locations, f.Location)
		}
	}

	items := make([]LootItem, 0, len(order))
	for _, fp := range order {
		it := byFP[fp]
		sort.Slice(it.Locations, func(i, j int) bool {
			a, b := it.Locations[i], it.Locations[j]
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			if a.Line != b.Line {
				return a.Line < b.Line
			}
			return a.Probe < b.Probe
		})
		items = append(items, *it)
	}

	sort.Slice(items, func(i, j int) bool {
		ri, rj := severityRank(items[i].Severity), severityRank(items[j].Severity)
		if ri != rj {
			return ri < rj
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Fingerprint < items[j].Fingerprint
	})

	assignIDs(items)
	return items
}

func hasLocation(locs []probes.Location, target probes.Location) bool {
	for _, l := range locs {
		if l == target {
			return true
		}
	}
	return false
}

// assignIDs derives an 8-hex-char ID ("L-xxxxxxxx") from the fingerprint and
// extends it on collision (decision 10).
func assignIDs(items []LootItem) {
	used := make(map[string]bool, len(items))
	for i := range items {
		hexPart := strings.TrimPrefix(items[i].Fingerprint, "sha256:")
		if len(hexPart) < 8 || !isHex(hexPart) {
			sum := sha256.Sum256([]byte(items[i].Fingerprint))
			hexPart = hex.EncodeToString(sum[:])
		}
		length := 8
		for length <= len(hexPart) {
			candidate := "L-" + hexPart[:length]
			if !used[candidate] {
				used[candidate] = true
				items[i].ID = candidate
				break
			}
			length += 2
		}
	}
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func severityRank(sev string) int {
	switch sev {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

// CalculateScore computes 100 - (critical*10 + high*4 + medium*2 + low*1),
// clamped to [0, 100], based on deduplicated loot items (SPEC.md).
func CalculateScore(loot []LootItem) int {
	penalty := 0
	for _, it := range loot {
		switch it.Severity {
		case "critical":
			penalty += 10
		case "high":
			penalty += 4
		case "medium":
			penalty += 2
		case "low":
			penalty += 1
		}
	}
	score := 100 - penalty
	if score < 0 {
		return 0
	}
	return score
}
