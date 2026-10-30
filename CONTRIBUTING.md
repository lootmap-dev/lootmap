# Contributing to lootmap

Thank you for helping developers see their laptops the way malware does!

## Ground Rules

1. **Never touch the real home directory in tests.** All file access goes through `internal/env.Env`. Tests must use `testdata/homes/*` or `t.TempDir()` via `env.NewFake`.
2. **Never commit real-format secrets.** Use `{{PLACEHOLDER}}` tokens in fixture files and generate fake secrets at runtime via `internal/testutil.FakeSecret`.
3. **No network calls.** Only `internal/osv` may import `net/http`.
4. **Tests first.** Every new probe, detector, or fixer requires table-driven tests and golden files.

---

## Add a New AI Tool in 20 Minutes

When a new AI coding agent or MCP client is released, you can add support in three small steps:

### Step 1: Add verified paths to `internal/probes/paths.yaml`

Add the macOS (`darwin`), `linux`, and `windows` relative home paths under `ai-transcripts` or `ai-config`:

```yaml
  ai-transcripts:
    darwin:
      - ".new-agent/sessions"
    linux:
      - ".config/new-agent/sessions"
    windows:
      - "AppData/Roaming/new-agent/sessions"
```

If you have not verified a path on a real installation of that OS, add `# UNVERIFIED` at the end of the line so reviewers know to verify it.

### Step 2: Add a template fixture in `testdata/homes/linux-dev/`

Create a sample transcript or config file using a `{{PLACEHOLDER}}` token (for example `{{OPENAI_KEY_1}}`)—never a real or realistic-looking key string.

### Step 3: Run tests and update goldens

```bash
LOOTMAP_UPDATE_GOLDEN=1 go test ./internal/probes/ -run TestAIProbesGoldenAndSQLite
go test ./...
golangci-lint run ./...
```

Inspect `git diff internal/probes/testdata/` to confirm the new location appears in the golden JSON with a `~/` shortened path and a masked preview.
