# Curl VPS installer

* **Author:** hsoni92
* **Date:** 2026-11-16

## Overview

An operator on a clean Ubuntu VPS runs one command (for example `curl -fsSL https://raw.githubusercontent.com/hsoni92/obelisk/main/install/install.sh | sudo bash`) to install the Obelisk platform. The command must not require `git clone` or a manual checkout step. The installer fetches a GitHub source archive (default branch at run time), unpacks it under `/opt/obelisk`, builds or provides the `obelisk` CLI, installs Docker when images are built on the host, and runs the existing `install/bootstrap.sh` with `OBELISK_PROFILE=vps` and `OBELISK_BUILD_IMAGES=true` on first boot.

Install completes when the platform Helm release is healthy and the admin UI is reachable through Kong on the VPS (same bar as `tests/integration/vm/smoke.sh`). Onboarding (GitHub sign-in, blueprint unlock, backups) is unchanged and happens after platform install. This entry path is VPS production only; local dev keeps Makefile / k3d flows.

## Analysis

### Install path and artifacts

Today’s real installer is `install/bootstrap.sh` (k3s, optional image build, config under `/etc/obelisk`, Helm, CLI install). `install/install.sh` does not exist yet. The curl entrypoint should stay thin: resolve ref, download archive, extract with correct layout (GitHub tarballs unpack to `obelisk-<ref>/`), set env, invoke bootstrap. Do not duplicate k3s or Helm logic in the wrapper.

### Distribution and “latest”

“Latest from GitHub” means the archive for the default branch (`main`) at install time unless `OBELISK_REF` is set (tag, branch name, or commit SHA). Production installs should document pinning with `OBELISK_REF` or a release tag when tags exist. A stable vanity URL (`https://…/install.sh`) can point at raw GitHub or a release asset later; v1 can document the raw URL on `main`.

### VPS success criteria and Kong exposure

Integration smoke passes only because the VM path sets `OBELISK_HELM_EXTRA_ARGS='--set kong.service.type=LoadBalancer'`. Default Helm values use ClusterIP for Kong, which is not reachable from the host IP on a bare VPS. The VPS installer must set the same LoadBalancer (k3s servicelb) default as the VM harness and print the reachable `http://<vps-ip>/` URL at the end.

### Prerequisites and time expectations

First install on a small VPS is dominated by on-host Docker image builds (integration docs cite roughly 15–25 minutes), not by downloading source. The brief does not promise a five-minute end-to-end install while `OBELISK_BUILD_IMAGES=true` remains the default. v1 ships build-on-host; prebuilt images in a registry are a follow-on to shorten install and simplify rollback.

The wrapper must ensure Docker is present before bootstrap when building images. The VM harness builds the CLI with `go build`; bootstrap only installs `/usr/local/bin/obelisk` if `$OBELISK_ROOT/obelisk` exists. The curl path must build the linux CLI on the host or ship a prebuilt binary in the archive.

### Security and supply chain

`curl | sudo bash` plus unpinned `get.k3s.io` and Helm’s install script expands trust surface. v1 should document download-inspect-run, optional archive checksum verification (`OBELISK_ARCHIVE_SHA256`), and keep config at `/etc/obelisk/obelisk.yaml` untouched on re-run (bootstrap already skips if present). Signed release bundles and dual channels (stable / canary) are a sensible v2 direction: thin immortal stub, fat versioned bundle, default stable for VPS.

### Re-run, upgrade, and observability

Re-running the curl command must be defined: first install vs upgrade (pull new ref, helm upgrade, avoid blind overwrite of operator edits under `/opt/obelisk`). Write install provenance (`/opt/obelisk/.install-meta.json` with ref and timestamp) and a persistent log (`/var/log/obelisk-install.log` or stage file) so failures are triaged without guessing.

### Codebase readiness

Bootstrap, `build-images.sh`, and Multipass/rsync integration already model the VPS spine. Missing pieces: `install/install.sh`, curl-shaped integration test, runbook/README updates, optional CI release job, Docker/CLI preflight in wrapper or bootstrap, Kong VPS defaults, and wiring `versions.lock` into k3s/Helm pins over time.

### Risk summary

| Risk | Mitigation (v1 / later) |
|------|-------------------------|
| Kong not reachable on VPS | VPS profile sets LoadBalancer; print URL |
| Long install / OOM on 8 GB | Honest docs; optional smoke tail; registry images later |
| No Docker / Go on bare host | Wrapper installs Docker; builds or bundles CLI |
| Moving `main` on prod | Document pin with `OBELISK_REF`; promote tags later |
| Supply chain | Checksum env; signed bundles deferred |

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Single documented curl one-liner for VPS production install (no `git clone` step for the operator).
  * Download GitHub archive into `/opt/obelisk`, run existing `bootstrap.sh` with `OBELISK_PROFILE=vps`.
  * Default `OBELISK_BUILD_IMAGES=true` on fresh VPS until prebuilt images exist.
  * `OBELISK_REF` override for pin by tag, branch, or SHA.
  * Kong reachable from host; admin UI HTTP smoke passes.
  * Update `docs/runbooks/install.md` and README so VPS prod source of truth is the curl path.
  * Integration test path that downloads an archive instead of only rsync from dev mount.

* **Out-of-Scope Non-Goals:**
  * Local dev / k3d / Multipass mount workflows as curl targets.
  * Tailscale on VPS (`OBELISK_TAILSCALE` stays disallowed for `vps`).
  * Mandatory container registry and GHCR publish in v1.
  * Full cosign/SBOM pipeline in v1 (document as follow-on).
  * `obelisk uninstall` and resumable stage machine beyond bootstrap’s current idempotency.
  * Replacing manual clone path for contributors (supported as fallback, not canonical for VPS).

## Alternatives Weighed

| Alternative | Outcome |
| --- | --- |
| **Keep git clone + manual `bootstrap.sh` as the only VPS path** | Rejected by author: operator must clone; conflicts with CUJ 1 / G5 and the stated goal of a one-line install. |
| **Shallow `git clone` inside `install.sh` instead of archive** | Rejected by author: still puts git on the critical path and reads as “clone”; archive matches “no checkout” wording. |
| **Monolithic remote script with all k3s/Helm logic inlined** | Rejected by author: duplicates `bootstrap.sh`, harder to test and drift-prone. |
| **Always track raw `main` with no pin story** | Rejected by author: unacceptable for production without documenting `OBELISK_REF` / future stable channel. |
| **Prebuilt images from registry as v1 requirement** | Rejected by author: large release/CI scope; defer until curl path ships; keep on-host build for now. |
| **apt/snap package as primary install** | Rejected by author: release and distro matrix overhead; curl matches design G5. |
| **Cloud-init / Terraform as primary VPS install** | Rejected by author: out of scope for indie VPS CUJ; curl remains the documented first step. |
| **Unsigned pipe-to-bash forever with no verify path** | Partially accepted: v1 keeps one-liner; must document inspect-before-run and optional checksum verify. |

## Key Decisions & Staging for Next Steps

**Locked for v1**

* VPS-only curl entry; local dev unchanged.
* Thin `install/install.sh` delegates to `install/bootstrap.sh`.
* Source delivery via GitHub archive API (not `git clone`).
* Default ref = default branch; override with `OBELISK_REF`.
* `OBELISK_ROOT=/opt/obelisk`, `OBELISK_PROFILE=vps`, first-boot `OBELISK_BUILD_IMAGES=true`.
* VPS Kong exposure via LoadBalancer (aligned with VM smoke).
* Canonical one-liner uses `sudo bash`.

**Open / staged**

1. Implement `install/install.sh` (fetch, extract, CLI build, Docker preflight, invoke bootstrap).
2. Harden bootstrap defaults for VPS Kong and Docker preflight when building images.
3. Add `tests/integration/vm/curl-install.sh` (or harness mode) without rsync mount.
4. Update runbook, README, and cross-link DR restore runbook to the same curl first step.
5. Optional: CI job publishing checksums and linux-amd64 CLI in release archives.
6. Follow-on: stable/canary channels, signed bundles, prebuilt images, `versions.lock` enforcement for k3s pin.

**Handoff:** Run `/egm-draft` with this brief when ready for a formal design doc under `docs/designs/`.