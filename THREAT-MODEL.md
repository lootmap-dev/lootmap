# Threat Model

`lootmap` is a local defensive tool that inspects sensitive developer files. Because any tool that reads credentials is itself a high-value target, `lootmap` is designed around strict least-privilege and supply-chain boundaries.

## 1. Assets Protected

1. **Developer secrets on disk:** Cloud credentials (`~/.aws`), Git/GitHub tokens, npm/PyPI tokens, SSH private keys, shell histories, `.env` files, and AI coding agent transcripts.
2. **Developer identity & privacy:** Hostnames, usernames, directory structures, and secret values must never leave the machine or appear in shareable reports.
3. **File integrity during `lootmap fix`:** Configuration files and AI transcripts must never be corrupted or truncated if a write is interrupted or if an AI agent is actively writing to a session file.

## 2. Trust Boundaries & Non-Goals

- **Same-user boundary:** `lootmap` runs as the normal desktop user without `sudo` or Administrator elevation. It does **not** defend against a kernel rootkit or an attacker who already has live code execution in the current user session.
- **Out of scope by design:** `lootmap` never decrypts browser password databases, OS keychains (macOS Keychain, Linux Secret Service, Windows Credential Manager), or cryptocurrency wallets.

## 3. Threats & Mitigations

| Threat | Mitigation in `lootmap` |
| --- | --- |
| **Accidental secret leakage** into terminal logs, CI logs, or shared JSON/Markdown reports | `internal/secret.Value` discards the raw string on creation, retaining only a SHA-256 fingerprint and masked preview (`ghp_••••••a9Xz`). `Chunk` masks `Text` on `String()`, `Format()`, and `MarshalJSON()`. Home paths are shortened to `~`. |
| **Compromised transitive dependency** attempting to phone home | Direct dependencies are kept under 10. `internal/archtest` and `depguard` forbid importing `net` or `net/http` outside `internal/osv`. Linux CI runs tests under `unshare --user --map-root-user --net`. |
| **Malicious or tampered `update-db` payload** | `internal/osv` verifies an Ed25519 signature (`crypto/ed25519`) over the downloaded `maldb.json` before writing to `~/.lootmap/maldb.json`. |
| **File corruption or race with a running AI coding agent** during `lootmap fix` | Live-session guard skips AI transcript files modified within the last 60 seconds (unless `--force` is passed). All writes go to a sibling temporary file, `fsync`, `chmod 0600`, and atomic `os.Rename`. |
| **Unmasked backup exposure** | Backups under `~/.lootmap/backups/<timestamp>/` are written with `0600` permissions and automatically pruned after 7 days. |
| **Tampered release binary** | Built on GitHub Actions with pinned commit SHAs, `CGO_ENABLED=0 -trimpath`, keyless Sigstore Cosign signatures, SPDX SBOMs, and SLSA Level 3 provenance. |
