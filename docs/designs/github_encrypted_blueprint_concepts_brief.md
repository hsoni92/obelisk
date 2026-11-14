# Encrypted GitHub blueprint mirror

* **Author:** hsoni92
* **Date:** 2026-11-14

## Overview

Obelisk operators want a durable, off-appliance copy of **desired state** (services, exposure, backup policy, `machine.id`) that survives a dead VPS without retyping settings. The proposed feature keeps **continuous sync to a private GitHub repo** as an **encrypted blueprint mirror**: non-sensitive structure in clear YAML, secrets in an encrypted bundle sealed with a **user-held passphrase** (never stored on disk). **S3/R2 stays the store for data backups** under `s3://<machine.id>/data/…`; backup endpoint and policy live in the blueprint so a fresh host can chain **settings from GitHub → data from S3** after unlock. Day-to-day, the appliance remains the writer; GitHub is a **write-only mirror**, not bidirectional GitOps.

## Analysis

### Authority and topology

Three stores exist; each must have one job or operators get split-brain on restore.

| Store | Holds | Written when | Wins on conflict |
| --- | --- | --- | --- |
| Local disk (`obelisk.yaml` + credential sidecars) | Live desired state | Every admin UI / API save | **During normal operation** |
| GitHub (encrypted mirror) | Snapshot of desired state + ciphertext secrets | After successful save, post-encrypt | **Only on explicit bootstrap / restore from GitHub** |
| S3/R2 (`machine.id` bucket) | Data backups; optional legacy `config/` tarball | Backup jobs | **For database and object data only** |

The approved `obelisk.md` CUJ 5 still describes `config/` on S3 as control-plane backup. This feature should **demote S3 `config/`** to optional fallback or derivative of the GitHub mirror once sync ships, and document that in CUJ 5. Running GitHub and S3 `config/` as co-equal control-plane sources without a winner breaks restore.

`machine.id` on a fresh host must **come from the decrypted blueprint before** `EnsureMachineIdentity` can assign a new ID; otherwise S3 data restore targets the wrong bucket.

### Encrypted blueprint mirror (not GitOps)

Auto-push on save is **not** cluster reconcile from git. There is **no silent pull** from GitHub while the appliance is running. GitHub edits apply only via an explicit **import / bootstrap / restore** flow. That removes merge races between UI and remote.

Pairing for headless VPS: **one-time setup from a laptop** (browser or CLI) that installs a **scoped machine credential** on the appliance (deploy key or GitHub App installation token in a K8s Secret). The VPS does not hold user OAuth refresh tokens for routine sync.

Sync pipeline after each `config.Save`: validate → assemble bundle → encrypt → pre-push plaintext scan → commit → push with **debounced coalescing** and **retry queue**. UI shows **saved locally / encrypt ok / pushed @ sha / failed (retrying)** separately from reconcile health.

Push failure must not corrupt local state: local yaml stays authoritative; queue retries with backoff; cap queue depth. Generation or monotonic `blueprint.generation` in spec rejects push if remote is unexpectedly ahead (clobber guard).

### User key and encryption envelope

Use a **Blueprint encryption passphrase** (KDF e.g. Argon2id, AEAD e.g. XChaCha20-Poly1305) distinct from **S3 upload credentials**. Passphrase unlocks a **bundle** (`manifest.json` + `secrets.enc` or single `blueprint.enc`), not necessarily every field of `obelisk.yaml` in git.

**Envelope pattern:** data encrypted with DEK; DEK wrapped by user key; `key_id` in manifest. Secret rotation re-wraps and pushes new HEAD; **do not rewrite git history** by default. Old commits remain old ciphertext (acceptable if operators understand rotation).

**Inventory required before build:** which fields stay cleartext in repo (service flags, backup endpoint, `machine.id`), which sidecars enter the bundle (`backup-remote`, mail, openrouter, instance credentials in yaml today), which stay host-only (GitHub machine token, ephemeral unlock).

**Key loss:** no server recovery. Document loudly at setup. Optional operator-printed **Recovery Kit** (repo URL, `machine.id`, S3 endpoint, key fingerprint) — not the key itself.

For auto-push, encryption uses a **session unlock**: operator unlocks blueprint crypto once (or on boot); derived key lives in memory with TTL — **not** persisting passphrase on disk. Re-lock blocks push until unlock (trade-off accepted vs prompting every save).

**v1 S3 plaintext** (G4-detail) remains a separate risk; long term, align **data blob encryption** with the same user key envelope so one passphrase story matches operator expectations.

### Chained rebirth (zero-effort target)

Target orchestrator phases (extend CUJ 5):

1. `pair_github` (once) — machine credential on appliance  
2. `pull_blueprint` — fetch encrypted mirror  
3. `unlock` — operator passphrase  
4. `apply_control` — write yaml + sidecars + K8s secrets  
5. `pull_data` / `restore_data` — S3 using backup creds from decrypted bundle  
6. `reconcile_services` — controller  
7. `exposure_safe` — Funnel off until re-enabled  

UI and CLI must **never** label step 3–4 alone as “restore complete.” Show **Settings restored** vs **Data restored** with next action.

Prefer a single orchestrated entry (`obelisk restore --chain` or **DR recover** wizard) over many undocumented verbs.

### Risk mitigations (operator-facing)

| Risk | Mitigation |
| --- | --- |
| Plaintext in git history | Encrypt-then-commit; pre-commit validation; bundle-only secrets |
| Push fails / GitHub down | Retry queue; local ops continue; degraded sync badge |
| Dual control-plane (Git vs S3 config) | GitHub mirror canonical for settings; retire or derive S3 `config/` |
| Headless auth | Laptop pairing CUJ; deploy key / App; device flow documented |
| Key loss | No recovery; Recovery Kit; S3 data may survive but secrets do not |
| Token compromise | Scope write to one repo; rotate without putting PAT in yaml |
| Save before reconcile | Push records desired state; status shows reconcile separately |
| Plaintext instance secrets in yaml today | Migration: bundle v1; grep/CI; move to encrypted payload |
| Stub backup worker | **Build order:** S3 backup/restore orchestrator before or in parallel with sync; do not ship sync-only vanity |

### Inventive synthesis

Name the pattern **Encrypted Blueprint Mirror** so operators do not expect Argo/Flux. GitHub gives **visibility and off-site config**; S3 gives **bulk data**. Unidirectional push eliminates GitOps conflict class. Laptop pairing inverts headless OAuth contradiction.

### Codebase readiness

No GitHub, crypto, or restore orchestrator code exists yet. Hooks: `config.Save`, `PUT /api/v1/config`, `EnsureMachineIdentity`, DR panel, `pkg/obelisk/spec.go`. Extend spec with `blueprint.github` (repo, branch, generation, sync_status, last_sha). New packages: crypto bundle, git transport, post-save sync worker.

**Implementation phases:** P0 design lock → P1 crypto bundle → P2 GitHub transport → P3 save hook → P4 bootstrap chain (needs restore orchestrator) → P5 UI/migration.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Private GitHub repo as **encrypted write-only mirror** of Obelisk desired state, including **backup remote policy** (endpoint, schedules, retention as spec fields).
  * **User-held passphrase** encrypts secret material; passphrase not stored on appliance.
  * **Auto-push** after successful spec saves (debounced, retried).
  * **Bootstrap / restore chain:** GitHub blueprint → unlock → reconcile → **S3 data restore** using decrypted backup credentials.
  * **Sync and restore status** in admin API/UI; headless pairing CUJ.
  * Explicit **mitigations** for push failure, clobber, key loss, and split DR messaging.

* **Out-of-Scope Non-Goals:**
  * Bidirectional GitOps (pull on timer, merge from GitHub while running).
  * Application deploys or Helm charts from git (unchanged product non-goals).
  * Database dumps or RustFS blobs in GitHub.
  * Multi-node or multi-writer appliances.
  * Mandatory GitHub for appliances that only use S3 DR (sync is opt-in).
  * Server-side recovery of lost user passphrase.

## Alternatives Weighed

| Alternative | Outcome |
| --- | --- |
| **S3-only encrypted control plane (extend G4 `config/` tarball)** | Rejected by author: wants GitHub ergonomics, visibility, and account-centric storage separate from backup bucket. |
| **Manual `obelisk blueprint push` only (no auto-sync)** | Rejected by author: wants continuous sync; manual export accepted only as emergency fallback. |
| **Cleartext `obelisk.yaml` in private repo, secrets only in S3** | Rejected: author wants secrets encrypted into blueprint with user key; also leaks topology in repo metadata. |
| **Defer GitHub until S3 DR is complete** | Partially accepted: reviewers insist **restore orchestrator and backup upload** must not lag far behind sync; shipping sync alone is unsafe. |
| **GitHub as sole DR (no S3)** | Rejected: data plane too large; contradicts existing `machine.id` bucket model. |
| **Full GitOps reconcile from git** | Rejected: conflicts with v1 non-goals and single-writer appliance model. |

## Key Decisions & Staging for Next Steps

**Locked (pending author confirm in draft):**

1. Pattern name: **Encrypted Blueprint Mirror**, unidirectional appliance → GitHub.  
2. **GitHub = control-plane mirror**; **S3 = data**; demote duplicate S3 `config/` authority in docs.  
3. **Opt-in** `blueprint.github.enabled` default false for existing installs.  
4. **Session unlock** for encryption to satisfy auto-push vs user-held key.  
5. **Machine credential** via pairing CUJ, not VPS OAuth per save.

**Resolved (2026-11-14):** D1 GitHub OAuth (+ device flow); D2 split bundle v1; D3 no S3 `config/` fallback; D4 full restore chain required for GA.

**Still open for draft:** instance credentials migration out of plaintext yaml; admin UI placement (Platform → Blueprint sync).

**Recommended build order:** Ship or finish **S3 backup upload + restore orchestrator skeleton** in the same release train as **P1–P3 sync**, so operators are not pushed to GitHub while data DR is still stub.