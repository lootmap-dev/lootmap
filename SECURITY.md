# Security Policy

## Supported Versions

| Version | Supported |
| --- | --- |
| `0.1.x` | Yes |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities in `lootmap`.

Instead, report vulnerabilities privately through [GitHub Security Advisories](https://github.com/lootmap-dev/lootmap/security/advisories/new).

Include:
1. Affected `lootmap` version and operating system.
2. Steps or minimal fixture reproducing the issue (never include real secrets).
3. Impact assessment (e.g., secret leakage in output, path traversal in backup/restore, signature verification bypass).

We acknowledge reports within **48 hours** and aim to release a signed fix within **7 days** for critical issues.

## Security Invariants Enforced in CI

- **Zero network by default:** `internal/archtest` and `golangci-lint` (`depguard`) block `net` and `net/http` imports everywhere except `internal/osv`. Linux CI runs the test suite inside `unshare --net`.
- **Zero raw secret leakage:** Raw secret strings are hashed and masked on construction inside `internal/secret.Value`. Every output format is tested against runtime-generated fake secrets.
- **Signed releases:** All release artifacts are built with `CGO_ENABLED=0 -trimpath`, accompanied by SPDX SBOMs, SLSA Level 3 provenance, and keyless Sigstore Cosign signatures.
