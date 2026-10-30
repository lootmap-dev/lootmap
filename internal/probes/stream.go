package probes

import (
	"bufio"
	"bytes"
	_ "embed"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/lootmap-dev/lootmap/internal/env"
)

//go:embed paths.yaml
var embeddedPaths []byte

type pathsDoc struct {
	Probes map[string]map[string][]string `yaml:"probes"`
}

var (
	pathsOnce sync.Once
	pathsMap  map[string]map[string][]string
)

// PathsFor returns the relative home paths configured in paths.yaml for the
// given probe ID and operating system.
func PathsFor(probeID, goos string) []string {
	pathsOnce.Do(func() {
		var doc pathsDoc
		if err := yaml.Unmarshal(embeddedPaths, &doc); err == nil {
			pathsMap = doc.Probes
		}
	})
	if pathsMap == nil {
		return nil
	}
	byOS, ok := pathsMap[probeID]
	if !ok {
		return nil
	}
	return byOS[goos]
}

// AnyPathExists reports whether at least one of the configured paths for
// probeID exists in e.FS().
func AnyPathExists(e env.Env, probeID string) bool {
	fsys := e.FS()
	for _, rel := range PathsFor(probeID, e.OS()) {
		if _, err := fs.Stat(fsys, rel); err == nil {
			return true
		}
	}
	return false
}

// StreamThresholdBytes is 50 MB (SPEC.md: files >50 MB are streamed in chunks,
// binary files are skipped).
const StreamThresholdBytes = 50 * 1024 * 1024

// IsBinarySniff inspects up to the first 512 bytes for a NUL byte.
func IsBinarySniff(sample []byte) bool {
	if len(sample) > 512 {
		sample = sample[:512]
	}
	return bytes.IndexByte(sample, 0) >= 0
}

// ScanTextFile reads rel from e.FS(). Binary files are skipped. Files larger
// than 50 MB (and all normal files) are streamed line by line with a bounded
// buffer so a 2 GB file never loads into memory at once.
func ScanTextFile(e env.Env, probeID, rel string, emit Emitter) error {
	f, err := e.FS().Open(rel)
	if err != nil {
		return nil // missing or unreadable files are skipped gracefully
	}
	defer func() { _ = f.Close() }()

	var head [512]byte
	n, err := f.Read(head[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return nil
	}
	if n == 0 {
		return nil
	}
	if IsBinarySniff(head[:n]) {
		return nil
	}

	reader := io.MultiReader(bytes.NewReader(head[:n]), f)
	shortPath := "~/" + strings.TrimPrefix(rel, "/")

	// If the whole content is a multiline PEM private key, emit the PEM block
	// as a single chunk so the multiline PEM regex matches it.
	if n < 512 && strings.Contains(string(head[:n]), "-----BEGIN ") && strings.Contains(string(head[:n]), "PRIVATE KEY-----") {
		all, err := io.ReadAll(io.LimitReader(reader, 64*1024))
		if err == nil {
			emit.EmitChunk(Chunk{
				Probe: probeID,
				Path:  shortPath,
				Line:  1,
				Text:  string(all),
			})
			return nil
		}
	}

	sc := bufio.NewScanner(reader)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	lineNo := 0
	var pemBuf strings.Builder
	pemStart := 0

	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if pemStart > 0 {
			pemBuf.WriteString("\n")
			pemBuf.WriteString(line)
			if strings.Contains(line, "-----END ") && strings.Contains(line, "PRIVATE KEY-----") {
				emit.EmitChunk(Chunk{
					Probe: probeID,
					Path:  shortPath,
					Line:  pemStart,
					Text:  pemBuf.String(),
				})
				pemBuf.Reset()
				pemStart = 0
			}
			continue
		}
		if strings.Contains(line, "-----BEGIN ") && strings.Contains(line, "PRIVATE KEY-----") {
			if strings.Contains(line, "-----END ") {
				emit.EmitChunk(Chunk{
					Probe: probeID,
					Path:  shortPath,
					Line:  lineNo,
					Text:  line,
				})
			} else {
				pemStart = lineNo
				pemBuf.Reset()
				pemBuf.WriteString(line)
			}
			continue
		}
		emit.EmitChunk(Chunk{
			Probe: probeID,
			Path:  shortPath,
			Line:  lineNo,
			Text:  line,
		})
	}
	return nil
}
