# Obelisk VPS uninstaller

* **Author:** hsoni92
* **Date:** 2026-11-16

## Overview

Operators on a production VPS need one command to tear down a broken first install or to reclaim the machine for reuse. The feature adds `sudo obelisk uninstall` and `install/uninstall.sh`, mirroring the install/update pattern. It removes **local** Obelisk state only: Kubernetes workloads in the `obelisk` namespace, Helm releases, PVCs, host paths, credentials, and on-host images. Remote blueprint repos, backup buckets, and GitHub OAuth apps are **not** touched. Uninstall supports `--dry-run`, interactive confirmation, and an optional flag to remove k3s. Dev (k3d, Multipass harness, local profile) is out of scope for v1.

## Analysis

**Why this exists.** Install leaves state across four layers (host tree, config/credentials, k3s/Helm, container images). Curl install and bootstrap can fail mid-way; today there is no symmetric teardown—only undocumented manual `helm uninstall` / `rm -rf` steps. Parent design (`obelisk.md`) already names `obelisk uninstall` for failed bootstrap, but implementation was deferred with the curl installer.

**Architecture mirror.** Follow `obelisk update` → `install/update.sh`: thin Go CLI (root check, env) execs `install/uninstall.sh` as the single source of truth. Bootstrap’s artifact list is the uninstall checklist inverted.

**Teardown order (critical).** Stop the controller first (scale to 0 or remove platform early) so reconcile does not recreate releases during uninstall. Then: enumerate and `helm uninstall` all releases in the namespace (services before or after platform—must discover dynamically via `helm list`, not a fixed type list). Delete PVCs when doing a full local wipe. Remove `obelisk-platform` and remaining namespace objects. Delete cluster-scoped **ClusterRole** / **ClusterRoleBinding** `obelisk-controller` (namespace delete does not remove these). Prune Obelisk-tagged images from k3s containerd; optional Docker layer prune for `obelisk-*` builds. Remove host paths. Remove `/usr/local/bin/obelisk` before or after tree—**delete `/opt/obelisk` last** so the script can finish while the binary still exists (or ship uninstall logic in a path that survives until the final stage).

**Full wipe vs shallow retry (resolved for v1).** The operator asked for **full local Obelisk wipe** as the primary behavior, with k3s optional. Goldfish reviews split on defaults: some argue a shallow `platform` tier for retry-after-failure, others warn that keeping PVCs undermines VPS reclaim. **v1 decision:** default `obelisk uninstall` performs **full local wipe** (cluster + PVCs + `/etc/obelisk` + `/var/lib/obelisk` + `/opt/obelisk` + CLI + images). Add **`--tier platform`** (or `--keep-data`) for “retry install without deleting PVCs or credential dirs” when the operator explicitly asks. k3s removal stays **off by default** via **`--remove-k3s`** with a second confirmation warning that all k3s workloads disappear, not only Obelisk.

**Dry-run contract.** Helm has no client dry-run for uninstall. `--dry-run` must inventory without mutating: Helm release names, PVC names/sizes, namespace, cluster RBAC names, host paths (existence), planned image removals, whether `--remove-k3s` would run. Emit human-readable lines and optional JSON for scripts. Prefix `[dry-run]` like install logging.

**Safety and observability.** Append stages to `/var/log/obelisk-uninstall.log`. Per-stage status: `ok | skip | fail`. Idempotent re-runs on partial state (missing k3s, no Helm, half-deleted tree). Exit code `2` on partial success with a **REMAINING:** summary. Uninstall does not call the admin API or wait on controller “uninstalling” service state—that is per-service Helm lifecycle, not host teardown.

**Out of install scope (do not remove in v1).** `docker.io`, snap Go, Helm installer, apt packages—install may add them; uninstall must not remove shared host packages.

**Inventive tension.** The product retains PVCs when a service is disabled; uninstall for reclaim is the opposite policy. The brief separates **disable** (keep data) from **uninstall** (reclaim disk) so operators are not surprised.

**Incomplete Goldfish lenses:** `egm-comprehension`, `egm-inventive-catalyst` were not run in this pass; re-audit at draft if needed.

## Goals vs. Non-Goals

**In-scope goals**

- VPS profile only; root-required `obelisk uninstall` and `install/uninstall.sh`.
- Full local Obelisk wipe by default (per operator intent); optional `--tier platform` / `--keep-data` for retry.
- `--dry-run`, interactive confirm (type `uninstall` or similar for destructive default), `--yes` for automation.
- Optional `--remove-k3s` after cluster cleanup (checkbox in wizard wording).
- Enumerate all Helm releases in `OBELISK_NAMESPACE` (default `obelisk`).
- Delete PVCs on default full wipe; purge credentials under `/var/lib/obelisk/credentials/`.
- Document manual fallback and failure recovery in `docs/runbooks/uninstall.md` (or install runbook section).
- Integration test: install → smoke → uninstall dry-run → uninstall → optional re-bootstrap smoke.

**Out-of-scope non-goals**

- k3d, Multipass harness helpers, `local` dev profile, Tailscale teardown.
- Remote GitHub blueprint delete, S3/R2 object delete, OAuth app revoke.
- Removing Docker engine or Go from the host.
- Resumable install stage machine (separate feature); uninstall may write `.uninstall-meta.json` but not full orchestrator parity.
- Curl-pipe uninstall one-liner (machine may lack tree); v2 if needed.
- Admin UI “uninstall appliance” button.

## Alternatives Weighed

| Alternative | Rejected because |
|-------------|------------------|
| Runbook-only manual steps | Fails dry-run, ordering, partial-install ergonomics, and reclaim safety. |
| Default shallow uninstall (keep PVCs/config) | Rejected by author: primary use is full reclaim and failed-install cleanup; shallow mode via explicit flag. |
| Controller-driven uninstall API | Reconcile races; overkill for v1; host/bash matches update. |
| `kubectl delete namespace` only | Hangs on finalizers; leaks cluster RBAC; misses host paths and images. |
| Remove k3s by default | Rejected: failed install retry and co-installed k3s workloads; opt-in only. |
| Separate Go implementation without `install/uninstall.sh` | Rejected: duplicates update pattern and leaves pre-CLI failed installs without a shell fallback. |

## Key Decisions & Staging for Next Steps

1. **Default behavior:** full local Obelisk wipe; `--tier platform` / `--keep-data` documented for retry.
2. **k3s:** `--remove-k3s` default false; second confirmation; run only after Obelisk namespace cleanup.
3. **Implementation:** `install/uninstall.sh` + `cmd/obelisk/uninstall_cmd.go`; shared path manifest with bootstrap (single list constant or sourced shell fragment to avoid drift).
4. **Ordering:** stop controller → helm uninstall all → PVC delete (if full) → namespace → cluster RBAC → images → host paths → `/opt/obelisk` last → optional k3s-uninstall.
5. **Tests:** `tests/integration/vm/uninstall-in-vm.sh`; dry-run + full + idempotent re-run required for merge.
6. **Handoff:** run `/egm-draft` on this brief to produce `docs/designs/obelisk_uninstaller.md` and wire CLI usage + runbook.