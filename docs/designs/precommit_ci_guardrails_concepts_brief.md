# Pre-commit and CI guardrails

* **Author:** hsoni92
* **Date:** 2026-11-16

## Overview

The public **obelisk** source repo needs automated checks before changes reach `main`, because strangers can fork it and VPS install pulls from `raw.githubusercontent.com`. Today there is no `.github/workflows` and no hook config; lint and tests run only when someone invokes `make` or npm manually. This brief defines a **thin local hook layer** (optional, fast) plus **mandatory GitHub Actions** as the real gate, including lint, build, tests, Helm chart lint, secret scanning, and light dependency hygiene.

This is **not** the same as the encrypted **blueprint mirror**: appliance `ScanPublicYAML` still blocks cleartext secrets at blueprint push time; repo gitleaks guards **committed** source, docs, and charts.

## Analysis

Goldfish review (zoom-out, design, implementation-readiness) converged on:

- **Scope the first slice to the design-doc CI item** (`obelisk.md` §18): one workflow reusing existing `Makefile` targets instead of duplicating commands.
- **Mandatory GHA; optional pre-commit.** Local hooks are bypassable (`--no-verify`); branch protection + required CI jobs protect `main` and PRs from forks.
- **One secret scanner:** gitleaks with a checked-in `.gitleaks.toml` allowlist for docs, Helm placeholder values, `*_test.go`, and schema fixtures. Do not merge appliance `ScanPublicYAML` rules into repo scanners.
- **False-positive risk is real** today (`github-blueprint.md` JSON examples, `scan_test.go` fake tokens, `changeme-in-ui` chart passwords). Allowlists must ship **before** required secret jobs.
- **Brownfield lint:** ESLint currently reports errors on several admin components; first CI rollout runs **web build + tests as blocking**, ESLint as **advisory** until fixed or baselined. Go: `go test`, `go build`, `go vet` before full golangci-lint enforcement.
- **Defer heavy or noisy checks:** Multipass VM tests, depcheck/knip, weekly schedules, golangci-lint-diff in hooks, dual gitleaks+detect-secrets.
- **Supply-chain angle:** protecting `install/**` on `main` matters as much as catching API keys in Go files.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * `.github/workflows/ci.yml` on push/PR: Go test, build both binaries, `go mod tidy` check, Helm lint, admin `npm ci` + build, full-repo gitleaks.
  * Optional `.pre-commit-config.yaml` + `make hooks-install` for staged gitleaks and basic file hygiene.
  * `.gitleaks.toml` allowlist policy documented and version-controlled.
  * `make ci` mirroring CI for local verification.
  * Pin Go 1.22, Node 20, Helm 3.14, gitleaks 8.21 in workflow (align `install/versions.lock` NEXTJS line with admin package).

* **Out-of-Scope Non-Goals:**
  * Integration VM harness in default PR CI.
  * Scanning gitignored operator paths (`dev/credentials/`, live `.env`).
  * Validating operator private blueprint repos or `secrets.enc` ciphertext.
  * Blocking merges on `npm audit` / govulncheck in v1 (advisory later).
  * Replacing or merging `internal/blueprint/ScanPublicYAML` into repo hooks.

## Alternatives Weighed

| Alternative | Why not (v1) |
|-------------|----------------|
| Pre-commit only, no CI | Bypassable; useless for fork PRs on a public repo. |
| detect-secrets + gitleaks | Dual config and baseline maintenance. Rejected by author: pick gitleaks. |
| golangci-lint in pre-commit on staged files | Slow, brittle on new branches; CI-only first. |
| depcheck/knip weekly | Noisy on single Next app; defer until second frontend or proven pain. |
| Reuse `ScanPublicYAML` for entire repo | False positives on tests and internal auth code. |

## Key Decisions & Staging for Next Steps

1. **Three layers:** (A) optional local pre-commit, (B) required GHA, (C) appliance blueprint pre-push scan — document separately in CONTRIBUTING and blueprint runbook.
2. **Phased enforcement:** merge workflow → observe → enable branch protection on `go`, `web`, `secrets`, `helm` → turn ESLint/golangci from advisory to required.
3. **Artifacts added in repo:** `.gitleaks.toml`, `.pre-commit-config.yaml`, `.github/workflows/ci.yml`, `CONTRIBUTING.md`, `make ci` / `make hooks-install`.
4. **Next:** `/egm-draft` this brief into a short design section or extend `obelisk.md` CI verification; fix ESLint errors to make `web-lint` required.