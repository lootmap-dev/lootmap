# Changelog

All notable changes to `lootmap` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - Unreleased

### Added
- **12 workstation probes:** `ai-transcripts`, `ai-config`, `cloud`, `git`, `npm`, `ssh`, `shell-history`, `shell-rc`, `dotenv`, `containers`, `iac`, and `packages`.
- **Secret detection engine:** Embedded `patterns.yaml` rules with Shannon entropy filtering, placeholder/example exclusion, SHA-256 fingerprint deduplication across files, and 0–100 security score (grades A–F).
- **Zero-leak architecture:** `internal/secret.Value` and masked `probes.Chunk` ensure raw secrets and usernames are never printed or serialized.
- **Reports:** 80-column `lipgloss` terminal loot map (`pretty`), shareable `json`, and `markdown` outputs, plus `lootmap explain <id>` and `lootmap explain --rotate`.
- **Safe remediation (`lootmap fix` & `lootmap undo`):** Dry-run preview, `0600` backups in `~/.lootmap/backups/` with 7-day auto-pruning, atomic file replacement, 60-second live AI session guard, and byte-for-byte restore.
- **Offline malicious package scanner (`lootmap check-packages` & `lootmap update-db`):** Embedded OSV `MAL-` database, Ed25519 signature verification, and parsers for `package-lock.json`, `pnpm-lock.yaml`, and `yarn.lock`.
