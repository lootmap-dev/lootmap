package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/engine"
	"github.com/lootmap-dev/lootmap/internal/loot"
)

//go:embed report.html.tmpl
var htmlTemplateSrc string

var htmlTemplate = template.Must(template.New("report").Parse(htmlTemplateSrc))

type htmlItem struct {
	ID        string
	Title     string
	Short     string
	Impact    string
	Severity  string
	Preview   string
	Locations []string
	AutoFix   bool
	Rotate    string
	Steps     []string
}

type htmlPlace struct {
	Path  string
	Items []htmlItem
	Worst string
}

type htmlData struct {
	Version   string
	OS        string
	Score     int
	Grade     string
	GradeTone string
	Critical  int
	High      int
	Medium    int
	Low       int
	Total     int
	Files     int
	AutoFix   int
	Rotate    int
	Items     []htmlItem
	Places    []htmlPlace
}

// WriteHTML renders a self-contained HTML report. The page loads nothing from
// the network (its Content-Security-Policy forbids it) and contains only the
// masked previews already present in res, so it is safe to open and share.
func WriteHTML(w io.Writer, res engine.Result) error {
	cat := loot.Default()
	d := htmlData{
		Version: res.Version,
		OS:      res.Host.OS,
		Score:   res.Score,
		Grade:   loot.Grade(res.Score),
		Total:   len(res.Loot),
	}
	switch d.Grade {
	case "A", "B":
		d.GradeTone = "good"
	case "C", "D":
		d.GradeTone = "warn"
	default:
		d.GradeTone = "bad"
	}

	places := map[string]*htmlPlace{}
	var order []string
	for _, it := range res.Loot {
		ent, _ := cat.Lookup(it.Kind)
		hi := htmlItem{
			ID:       it.ID,
			Title:    ent.Title,
			Short:    ent.Short,
			Impact:   it.Impact,
			Severity: it.Severity,
			Preview:  it.Preview,
			AutoFix:  autoFixable(it),
			Rotate:   it.Rotate,
			Steps:    ent.Steps,
		}
		if hi.Impact == "" {
			hi.Impact = ent.Impact
		}
		for _, l := range it.Locations {
			loc := l.Path
			if l.Line > 0 {
				loc = fmt.Sprintf("%s:%d", l.Path, l.Line)
			}
			hi.Locations = append(hi.Locations, loc)
			p, ok := places[l.Path]
			if !ok {
				p = &htmlPlace{Path: l.Path}
				places[l.Path] = p
				order = append(order, l.Path)
			}
			p.Items = append(p.Items, hi)
		}
		switch it.Severity {
		case "critical":
			d.Critical++
		case "high":
			d.High++
		case "medium":
			d.Medium++
		default:
			d.Low++
		}
		if hi.AutoFix {
			d.AutoFix++
		}
		if it.Rotate != "" {
			d.Rotate++
		}
		d.Items = append(d.Items, hi)
	}

	sort.Strings(order)
	for _, path := range order {
		p := places[path]
		p.Worst = worstSeverity(p.Items)
		d.Places = append(d.Places, *p)
	}
	d.Files = len(d.Places)
	return htmlTemplate.Execute(w, d)
}

func worstSeverity(items []htmlItem) string {
	rank := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1}
	worst := "low"
	for _, it := range items {
		if rank[it.Severity] > rank[worst] {
			worst = it.Severity
		}
	}
	return strings.ToLower(worst)
}
