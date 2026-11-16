# Backup lineage (first-class backups)

* **Author:** hsoni92
* **Date:** 2026-11-16
* **Status:** RFC (brainstorm published for team feedback)

## Overview

Obelisk today stores backup data as timestamped objects under `data/{type}/{instance_id}/` in the machine bucket. The admin UI shows policy, destinations, last run time per instance, and machine-wide restore. There is no backup id, no history browser, no point-in-time restore for one service, and no record of what policy was in force when a dump was taken.

This feature makes each successful upload a **backup record** the operator can name, browse, restore, copy into a new service, and delete on purpose. Records belong to one service instance at a time and form a **lineage** inside that service: new backups link to a parent (usually the previous successful node). **Deploy from backup** creates a **new** service with definition and data; that service starts a **new** lineage root. The old service’s backups may become **orphans** if the service is removed, unless the operator opts in to delete them.

The primary consumer is the **admin console**. The UX goal is a small number of clicks for the common path (create backup, restore the active snapshot). History can be shown as a horizontal tree per service; destructive actions need stronger guards.

## Analysis

Today’s engine uploads dumps after `POST /api/v1/backup/run`, tracks `last_backup_by_instance` in `backup-status.json`, and applies **replication factor (RF)** as “copy this run to the first N healthy enabled destinations.” `data_retention_days` appears in policy but is **not enforced** in code. Scheduled backups are described in design docs but are still a stub (manual run only). Machine restore downloads all of `data/`, not one chosen snapshot.

### Two layers: lineage record and object-store blob

A backup **node** is not the same thing as an S3 key. Each node has metadata (id, parent within the service, timestamp, trigger, size, RF at capture, retention witness, tags, link to object keys). The **blob** lives in the bucket as today or in a versioned layout with a **manifest sidecar** (`manifest.json`) written at upload time. The UI and API read an **index** built from manifests and listing, not from guessing filenames alone. **Cascade delete** removes metadata and objects on all enabled destinations. **Orphan** means the instance row is gone or disabled but backup records and blobs remain.

### Lineage rules (aligned with author decisions)

| Rule | Behavior |
|------|----------|
| Scope | One forest per service instance. No shared tree across services. |
| Parent link | Each new successful backup is a child of the prior successful node in that service (time order). Manual and scheduled runs differ only by `trigger`. |
| Active pointer | Each service points at one **active** backup (default target for the fast restore path; default parent for the next backup unless repointed). |
| Deploy from backup | New instance id, new lineage root, `source_backup_id` is provenance only (no cross-service graph edge). RF at bootstrap copies from source unless overridden before first run. |
| Service delete | Checkbox **Also delete backups** (default off). Unchecked → lineage marked orphan, nodes greyed in UI, blobs kept until retention or explicit delete. |
| Delete backup node | **Cascade** to all descendants in that service’s forest. Typed confirmation required. |

### Restore vs deploy

| Action | Target | Lineage | Identity |
|--------|--------|---------|----------|
| **Restore** | Same live or disabled instance | Continues | Same instance id |
| **Deploy from backup** | New instance | New root forest | New id; def + data from backup witness |

Restore is destructive for database types (stop service, replace data). **Three clicks** applies only to **restore active backup** on an existing instance after pre-flight checks. Picking an older node or deploy-from-backup adds confirmation steps.

### Policy: one editor, witness on each node

Global `backup.policy` and per-instance `policy_override` in `obelisk.yaml` remain the **only** place operators edit schedule, RF, and retention. Backup nodes store **capture-time witness** fields (read-only). Changing policy affects the next run, not re-upload of old blobs. **Set RF on a node** in the UI means editing effective policy for descendants on bootstrap, not retroactive multi-destination backfill in v1.

### Platform DR vs per-service lineage

The DR tab keeps machine-wide destinations, policy, **Backup now**, and chained restore for a dead appliance. Per-service lineage lives under **Services** (card actions + detail tree). Cross-links between the two avoid duplicate “where do I click?” confusion.

### Index authority and migration

Recommended v1 authority: **manifest sidecar per backup** plus a machine-local index file under credentials (e.g. `backup-index.json`) updated after successful upload, rebuilt from S3 on upgrade via inventory pass. Legacy objects without manifests get synthetic ids from timestamp in the key and `parent_id` null or inferred order.

Stable **instance id** collision is a risk if an operator deletes `postgres/foo` and later creates another `postgres/foo`. The brief should either introduce an immutable `lineage_id` per forest or document that history is keyed by object prefix and may collide when ids are reused.

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Index and S3 drift | Tombstone-first deletes; reconcile job; health flag on backup config |
| Unbounded orphans + no retention | Enforce `data_retention_days` before scaling history UI; DR summary of orphan bytes |
| Cascade delete mistakes | Typed confirm; show child count and sizes |
| Per-backup RF confusion | Label as destination copy count; v1 witness-only on nodes |
| Scheduled backup stub | Lineage UI is thin until cron/reconciler exists; brief stages runner with catalog |
| Restore not implemented per type | Postgres first; other engines follow; block restore in UI until supported |
| ListObjects cost | Paginated API; cache index; lazy metadata on node select |

### Ideal final result (inventive synthesis)

Operator opens a service, sees **active backup** and time, restores in three clicks when safe. History is a **per-service** timeline or horizontal tree. Deploy-from-backup starts a new root elsewhere with a breadcrumb to the source. Policy is edited once in yaml; each backup shows what applied at capture. Service delete preserves orphans by default; retention caps storage; cascade delete is only when removing a backup subtree.

### Staging disagreement (review lenses)

**Zoom-out / YAGNI** argued for deferring first-class entities and shipping only a read-only S3 catalog on DR. **Design / codebase / readiness** agreed the direction is right but said a **list + manifest + single-service restore** must land before a polished tree, deploy-from-backup, and cascade delete. This brief keeps the **author’s entity model** and sequences work so v1 value is not blocked on horizontal tree polish.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Backup records with stable ids, parent links within one service, active pointer, capture-time witness (RF, retention, trigger).
  * Manifest sidecar at upload and durable index for admin API and UI.
  * Per-service history UI: timeline list minimum; horizontal tree on service detail when data supports it.
  * One-click **Create backup** from service card (single-instance run or filtered batch).
  * **Restore active backup** on same instance with guarded fast path (postgres first).
  * **Deploy from backup** → new service, def + data, new lineage root, inherited RF unless overridden.
  * Service delete dialog with opt-in **Also delete backups** (default off); orphan display (greyed).
  * Cascade delete of a backup node and its descendants within one forest.
  * Backfill index from existing `data/` keys on upgrade.
  * Retention enforcement tied to policy (prune eligible nodes; protect active pointer).
  * Clear separation from machine-wide DR restore.

* **Out-of-Scope Non-Goals:**
  * Cross-service lineage graph or restore-in-place into a different instance id.
  * Editable per-backup RF/retention/tags that re-upload or override global policy retroactively (v1).
  * Public automation API / CLI parity as a hard v1 requirement (admin console first).
  * Redis and non–data-service types in backup lineage (existing `BackupDataServiceTypes` only).
  * Control-plane restore from S3 `config/`; blueprint stays settings authority.
  * Automatic orphan purge on service delete without explicit opt-in.
  * Cross-`machine.id` backup adoption without DR copy story.
  * Full point-in-time restore for every engine in the first release slice.

## Alternatives Weighed

| Alternative | Rejected because |
|-------------|------------------|
| S3 LIST-only UI with no index | Slow, capped listing, no parent/active/orphan semantics; rejected by Goldfish lenses. |
| Flat DR catalog only, no first-class ids | Does not meet author goal of lineage, deploy-from-backup, or cascade delete; rejected by author. |
| Lineage index only in `obelisk.yaml` | Unbounded git-tracked yaml; conflicts with blueprint sync; rejected by implementation-readiness. |
| SQLite as sole authority | Survives poorly on fresh DR host without rebuild from S3; rejected for v1 sole store (cache later ok). |
| Shared mega-tree across all services | Contradicts per-service forest after bootstrap; rejected by author. |
| Deploy-from-backup as alias of machine `restore/chain` | Wrong scope (whole bucket vs one snapshot + new instance); rejected by design review. |
| Ship horizontal tree before list + restore | UI without restore job is risky; rejected for staging order, not for end-state UI. |
| Defer first-class entities entirely | Rejected by author; catalog alone does not deliver lineage actions. |

## Key Decisions & Staging for Next Steps

**Decisions locked in ideation**

1. Admin console is the v1 consumer.
2. One consistent point-in-time per node; manual vs scheduled = trigger only.
3. Parent chain within a service follows creation order (not only restore edges).
4. Deploy-from-backup copies service definition + data; **new lineage root**; `source_backup_id` metadata only.
5. Service delete: orphans by default; optional delete all backups.
6. Cascade delete on backup node includes descendants in that forest.
7. RF on nodes = witness of destination copy count at capture; policy edits apply forward.

**Decisions still required before `/egm-draft`**

| Topic | Options to nail in draft |
|-------|---------------------------|
| Backup id | UUID vs deterministic from key |
| Object layout | Keep `timestamp` keys + sidecar vs `/{backup_id}/` prefix |
| Lineage key | Reused instance id vs immutable `lineage_id` |
| Index location | `credentials/backup-index.json` vs S3 aggregate manifest |
| Tree insertion edge cases | Concurrent runs; failed upload nodes |
| Orphan retention | Same `data_retention_days` vs grace multiplier |

**Proposed build order**

1. **P0 — Schema:** manifest format, index shape, legacy backfill spec.
2. **P1 — Write path:** register node on successful upload; enforce retention prune (dry-run first).
3. **P2 — Read API + UI list:** per-service history, metadata card, active pointer display.
4. **P3 — Restore active (postgres):** async job, confirmations, reconcile integration.
5. **P4 — Orphans + service delete hook:** grey state, opt-in backup delete.
6. **P5 — Deploy-from-backup + cascade delete:** new instance flow, guarded subtree delete.
7. **P6 — Horizontal tree presentation** (if list + jobs stable); scheduled backup runner in parallel when ready.

**Handoff:** After team feedback, run `/egm-draft` on `docs/designs/backup_lineage_concepts_brief.md` → formal `docs/designs/backup_lineage.md` design. Related context: `docs/designs/obelisk.md` (CUJ 4–5), `pkg/obelisk/backup_scope.go`.