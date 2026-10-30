package osv

import (
	"encoding/json"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// PackageRef is a single resolved package name and version found in a lockfile
// or package cache.
type PackageRef struct {
	Name    string
	Version string
	Line    int
}

// ParsePackageLock parses npm package-lock.json (v1, v2, and v3).
func ParsePackageLock(data []byte) []PackageRef {
	var doc struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
		Packages map[string]struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}

	seen := map[string]bool{}
	var out []PackageRef
	add := func(name, ver string) {
		name = strings.TrimSpace(name)
		ver = strings.TrimSpace(ver)
		if name == "" || ver == "" {
			return
		}
		key := name + "@" + ver
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, PackageRef{Name: name, Version: ver, Line: 1})
	}

	for name, dep := range doc.Dependencies {
		add(name, dep.Version)
	}
	for pkgPath, pkg := range doc.Packages {
		if pkgPath == "" {
			continue
		}
		name := pkg.Name
		if name == "" {
			if idx := strings.LastIndex(pkgPath, "node_modules/"); idx >= 0 {
				name = pkgPath[idx+len("node_modules/"):]
			} else {
				name = pkgPath
			}
		}
		add(name, pkg.Version)
	}
	return out
}

// ParsePNPMLock parses pnpm-lock.yaml (v6 and v9 formats).
func ParsePNPMLock(data []byte) []PackageRef {
	var doc struct {
		Packages map[string]any `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	var out []PackageRef
	seen := map[string]bool{}
	for key := range doc.Packages {
		name, ver := splitPNPMKey(key)
		if name == "" || ver == "" {
			continue
		}
		k := name + "@" + ver
		if !seen[k] {
			seen[k] = true
			out = append(out, PackageRef{Name: name, Version: ver, Line: 1})
		}
	}
	return out
}

func splitPNPMKey(k string) (string, string) {
	k = strings.TrimPrefix(k, "/")
	if idx := strings.IndexByte(k, '('); idx > 0 {
		k = k[:idx]
	}
	// Handle /@scope/pkg@1.2.3 or /@scope/pkg/1.2.3 or pkg@1.2.3
	at := strings.LastIndexByte(k, '@')
	if at > 0 {
		return k[:at], k[at+1:]
	}
	slash := strings.LastIndexByte(k, '/')
	if slash > 0 {
		return k[:slash], k[slash+1:]
	}
	return "", ""
}

var (
	yarnHeaderRe  = regexp.MustCompile(`^"?(@?[^"@\s]+)@[^":]+`)
	yarnVersionRe = regexp.MustCompile(`^\s+version\s+"?([^"\s]+)"?`)
)

// ParseYarnLock parses classic and Berry yarn.lock files.
func ParseYarnLock(data []byte) []PackageRef {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var currentPkg string
	var currentLine int
	var out []PackageRef
	seen := map[string]bool{}

	for idx, raw := range lines {
		lineNo := idx + 1
		if len(raw) > 0 && raw[0] != ' ' && raw[0] != '\t' && raw[0] != '#' {
			if m := yarnHeaderRe.FindStringSubmatch(raw); len(m) == 2 {
				currentPkg = m[1]
				currentLine = lineNo
			}
			continue
		}
		if currentPkg != "" {
			if m := yarnVersionRe.FindStringSubmatch(raw); len(m) == 2 {
				ver := m[1]
				k := currentPkg + "@" + ver
				if !seen[k] {
					seen[k] = true
					out = append(out, PackageRef{Name: currentPkg, Version: ver, Line: currentLine})
				}
				currentPkg = ""
			}
		}
	}
	return out
}
