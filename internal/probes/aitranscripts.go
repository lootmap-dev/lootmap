package probes

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/lootmap-dev/lootmap/internal/env"
)

// AITranscriptsProbe scans session transcripts and chat histories from Claude
// Code, Codex CLI, Gemini CLI, Cursor, Windsurf, and GitHub Copilot Chat.
type AITranscriptsProbe struct{}

func (AITranscriptsProbe) ID() string               { return "ai-transcripts" }
func (AITranscriptsProbe) Available(e env.Env) bool { return AnyPathExists(e, "ai-transcripts") }

func (AITranscriptsProbe) Scan(ctx context.Context, e env.Env, emit Emitter) error {
	fsys := e.FS()
	for _, root := range PathsFor("ai-transcripts", e.OS()) {
		st, err := fs.Stat(fsys, root)
		if err != nil {
			continue
		}
		if !st.IsDir() {
			if err := scanTranscriptFile(ctx, e, root, emit); err != nil {
				return err
			}
			continue
		}
		_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if isTranscriptCandidate(p) {
				_ = scanTranscriptFile(ctx, e, p, emit)
			}
			return nil
		})
	}
	return nil
}

func isTranscriptCandidate(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".jsonl", ".json", ".log", ".txt", ".md", ".vscdb", ".sqlite", ".db":
		return true
	}
	return false
}

func scanTranscriptFile(ctx context.Context, e env.Env, rel string, emit Emitter) error {
	ext := strings.ToLower(path.Ext(rel))
	if ext == ".vscdb" || ext == ".sqlite" || ext == ".db" {
		return scanSQLiteStrings(ctx, e, "ai-transcripts", rel, emit)
	}
	return ScanReaderStream(ctx, e, "ai-transcripts", rel, emit)
}

// ScanReaderStream streams a potentially huge file (e.g. 2 GB JSONL transcript)
// in 64 KB chunks without holding the whole file in memory.
func ScanReaderStream(ctx context.Context, e env.Env, probeID, rel string, emit Emitter) error {
	f, err := e.FS().Open(rel)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	br := bufio.NewReaderSize(f, 128*1024)

	peek, err := br.Peek(512)
	if len(peek) == 0 && err != nil {
		return nil
	}
	if IsBinarySniff(peek) {
		return nil
	}

	lineNo := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slice, isPrefix, err := br.ReadLine()
		if len(slice) > 0 {
			if !isPrefix {
				lineNo++
			} else if lineNo == 0 {
				lineNo = 1
			}
			// Fast pre-filter: skip scanning lines that cannot possibly contain
			// any secret marker or prefix.
			text := string(slice)
			if mightContainSecret(text) {
				emit.EmitChunk(Chunk{
					Probe: probeID,
					Path:  shortPath,
					Line:  lineNo,
					Text:  text,
				})
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return nil
		}
	}
}

// mightContainSecret is a fast ASCII pre-filter that avoids running 13 regexes
// on millions of benign transcript lines during multi-GB scans.
func mightContainSecret(s string) bool {
	if len(s) < 12 {
		return false
	}
	for _, needle := range []string{
		"ghp_", "github_pat_", "gho_",
		"AKIA", "ASIA",
		"npm_", "pypi-",
		"sk-", "sk_live_", "rk_live_",
		"xox", "AIza",
		"PRIVATE KEY",
		"://",
	} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

// scanSQLiteStrings extracts printable ASCII runs from a SQLite/vscdb file
// (such as Cursor or VS Code state.vscdb) in streaming blocks.
func scanSQLiteStrings(ctx context.Context, e env.Env, probeID, rel string, emit Emitter) error {
	f, err := e.FS().Open(rel)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	shortPath := "~/" + strings.TrimPrefix(rel, "/")
	buf := make([]byte, 64*1024)
	var carry []byte
	blockNo := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := f.Read(buf)
		if n > 0 {
			blockNo++
			chunk := append(carry, buf[:n]...)
			printable := extractPrintableLines(chunk)
			for _, line := range printable {
				if mightContainSecret(line) {
					emit.EmitChunk(Chunk{
						Probe: probeID,
						Path:  shortPath,
						Line:  blockNo,
						Text:  line,
					})
				}
			}
			if len(chunk) > 256 {
				carry = append([]byte(nil), chunk[len(chunk)-256:]...)
			} else {
				carry = append([]byte(nil), chunk...)
			}
		}
		if err != nil {
			break
		}
	}
	return nil
}

func extractPrintableLines(b []byte) []string {
	var out []string
	var cur bytes.Buffer
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur.WriteByte(c)
		} else if cur.Len() > 0 {
			if cur.Len() >= 12 {
				out = append(out, cur.String())
			}
			cur.Reset()
		}
	}
	if cur.Len() >= 12 {
		out = append(out, cur.String())
	}
	return out
}
