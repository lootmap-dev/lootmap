# lootmap

**See your laptop the way malware does. Then fix it.**

<!-- GIF Placeholder: 15-second terminal recording showing `lootmap scan` revealing the loot map (Score 20/100 F) followed by `lootmap fix` raising the score to B. -->
![lootmap demo placeholder](docs/demo-placeholder.gif)

---

## Install

### Homebrew (macOS & Linux)

```bash
brew install lootmap-dev/tap/lootmap
```

### Scoop (Windows)

```powershell
scoop bucket add lootmap https://github.com/lootmap-dev/scoop-bucket
scoop install lootmap
```

### Signed Release Binary (macOS, Linux, Windows)

Download the archive, `.sig`, and `.pem` from [GitHub Releases](https://github.com/lootmap-dev/lootmap/releases) and verify the keyless Sigstore Cosign signature before running:

```bash
cosign verify-blob \
  --certificate-identity-regexp="https://github.com/lootmap-dev/lootmap/.github/workflows/release.yml@refs/tags/v.*" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  checksums.txt
```

### Go Install

```bash
go install github.com/lootmap-dev/lootmap/cmd/lootmap@latest
```

---

## Why lootmap exists

1. **Shai-Hulud npm worm ([MITRE ATT&CK S9008](https://attack.mitre.org/software/S9008/), [Hackmag](https://hackmag.com/news/shai-hulud-2-0)):** Executed on package install and harvested `~/.aws/credentials`, `~/.npmrc`, GitHub tokens, and SSH private keys across more than 25,000 developer machines in three days.
2. **Bitwarden CLI supply-chain hijack ([JFrog Research](https://research.jfrog.com/post/bitwarden-cli-hijack/)):** Targeted cloud and git credentials alongside AI coding assistant and MCP server configuration files.
3. **Plaintext AI agent transcripts ([eSEOspace](https://eseospace.com/blog/claude-code-secrets-management/), [DEV](https://dev.to/beydemirfurkan/i-scanned-my-own-laptop-for-leaked-secrets-and-found-62311-of-them-3gp6)):** Coding agents log every file and tool output they read to disk in plaintext (`~/.claude/projects/`, `~/.codex/sessions/`, Cursor `state.vscdb`), leaving thousands of live secrets waiting for any infostealer.

---

## Privacy & Trust Guarantees

> **Zero network by default · Zero telemetry · Zero secret storage · Zero sudo**
>
> - **No network access:** `lootmap scan`, `lootmap fix`, `lootmap doctor`, and `lootmap check-packages` never open a socket. Importing `net/http` outside `internal/osv` is blocked in CI.
> - **Raw values are never printed or saved:** Matched secrets are immediately hashed (`sha256:…`) and masked (`ghp_••••••a9Xz`) inside `internal/secret.Value`. No raw secret ever touches stdout, stderr, logs, or JSON/Markdown reports.
> - **No browser or keychain harvesting:** `lootmap` never reads browser password databases, OS keychains, or crypto wallets.
> - **No root/admin privileges:** Runs strictly as your normal user on files you already own.

---

## What lootmap scans (12 Probes)

**Supported AI Agents:** `Claude Code` · `Claude Desktop` · `OpenAI Codex CLI` · `Gemini CLI` · `Cursor` · `Windsurf` · `GitHub Copilot Chat` · `VS Code MCP`

| Probe | Where it looks | What it finds | Severity | Auto-fix (`lootmap fix`) |
| --- | --- | --- | --- | --- |
| `ai-transcripts` | `~/.claude/projects/**/*.jsonl`, `~/.codex/sessions/`, Gemini CLI, Cursor, Windsurf, Copilot Chat | Live keys leaked into chat/tool history | Critical | Redacts in place (`[REDACTED:<kind>]`) with backup |
| `ai-config` | `~/.claude.json`, `claude_desktop_config.json`, `~/.cursor/mcp.json`, `.codex/config.toml`, VS Code MCP | Plaintext tokens in MCP server definitions | High | Tightens permissions to `0600` + vault guide |
| `cloud` | `~/.aws/credentials`, `~/.aws/config`, `~/.config/gcloud/`, `~/.azure/` | Long-lived AWS IAM keys, GCP service account JSONs | Critical | Rotation & AWS SSO migration guide |
| `git` | `~/.git-credentials`, `~/.gitconfig`, `~/.config/git/config` (+ `include.path`), `~/.config/gh/hosts.yml` | `credential.helper = store`, plaintext PATs | Critical | Switches helper to `osxkeychain` / `libsecret` / `manager` |
| `npm` | `~/.npmrc`, `~/.pypirc` | Publish tokens; `ignore-scripts=false` | Critical | Sets `ignore-scripts=true` + token rotation guide |
| `ssh` | `~/.ssh/` | Passphraseless private keys, `ForwardAgent yes`, loose permissions | High | Tightens permissions to `0600`, sets `ForwardAgent no` |
| `shell-history` | `.zsh_history`, `.bash_history`, `fish_history`, PowerShell `ConsoleHost_history.txt` | Secrets pasted into shell commands | High | Redacts in place with backup |
| `shell-rc` | `~/.zshrc`, `~/.bashrc`, `~/.profile`, `config.fish`, PowerShell profiles | `export OPENAI_API_KEY=…` in startup scripts | High | Vault reference guide |
| `dotenv` | `~/code`, `~/projects`, `~/dev`, `~/src`, `~/Documents/GitHub` (or `--roots`) | Production secrets & DB URLs in `.env*` files | High | Remediation guide |
| `containers` | `~/.docker/config.json`, `~/.kube/config` | Base64 registry auths without `credsStore`, kubeconfig tokens | High | Tightens permissions to `0600` |
| `iac` | `~/.terraform.d/credentials.tfrc.json`, `~/.pulumi/credentials.json` | Terraform Cloud and Pulumi tokens | Medium | Tightens permissions + rotation guide |
| `packages` | Project lockfiles (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`) & npm cache | Known malicious package versions (`OSV MAL-` offline DB) | Critical | Incident response steps |

---

## Honest Comparison

| Feature | **lootmap** | [bagel](https://boostsecurityio.github.io/bagel) | TruffleHog / Gitleaks | [agentsweep](https://pypi.org/project/agentsweep/) |
| --- | --- | --- | --- | --- |
| Shows attacker blast radius ("what you lose") | **Yes** | No (inventory list) | No (raw regex hits) | No |
| Safe auto-remediation (`fix` + `undo`) | **Yes** | No | No | AI transcripts only |
| Covers AI transcripts + MCP + Cloud + SSH + Git + npm | **Yes (12 probes)** | Partial | Unstructured file walk | AI transcripts only |
| Offline malicious package check (`OSV MAL-`) | **Yes** | No | No | No |
| Single static Go binary, zero runtime deps | **Yes** | Yes | Yes | No (Python/pip) |
| License | **Apache-2.0** | GPL-3.0 | AGPL-3.0 / MIT | MIT |

---

## Contributing — Add a New AI Tool in 20 Minutes

Adding support for a new AI coding assistant only takes a path entry in `internal/probes/paths.yaml` and a fixture test under `testdata/homes/`. See [CONTRIBUTING.md](CONTRIBUTING.md) for the step-by-step guide.

## License

Apache-2.0 — see [LICENSE](LICENSE).
