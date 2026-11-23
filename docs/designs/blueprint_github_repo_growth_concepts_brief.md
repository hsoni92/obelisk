# Blueprint GitHub repo growth (retained daily history)

* **Author:** hsoni92
* **Date:** 2026-11-23

## Review status

Parallel Goldfish reviewers did not run (host usage limit). This brief is synthesized from the anchored hypothesis, your correctness constraint, and inspection of `internal/blueprint/github.go` and `internal/blueprint/sync.go`. Re-run `/egm-brainstorm` Goldfish review when subagents are available if you want independent lens critique before draft.

## Overview

Operators sync encrypted appliance settings to a private GitHub repository (default `obelisk-appliance`), often one branch per machine. Every config save triggers a debounced push that appends another commit. Over months the branch and repo object set grow without bound.

The change caps growth while **keeping git history**: each machine branch keeps a rolling window of **at most one commit per calendar day (UTC) for the last 30 days**, and the **branch tip is always the latest successful snapshot** (today’s tip updates on every push that day). Older commits fall off the branch and become unreachable until GitHub GC. Each retained commit must remain a valid, decryptable bundle. Generation in `manifest.json` stays the authority for import and conflict rules. Post-push verify (fetch by commit SHA, decrypt) must still gate updates to local generation and `last_commit_sha`.

## Analysis

### Why the repo grows today

`GitHub.Push` always links `parents: [previous HEAD]`, so multiple saves on the same day add many commits. Blobs are re-uploaded each time. Nothing trims ancestors.

### Retention policy (operator requirement)

| Rule | Meaning |
| --- | --- |
| Latest always | Branch tip = most recent verified push; default import uses this |
| One per day | Multiple pushes on the same UTC day **replace** that day’s commit (no intraday chain growth) |
| 30-day window | Commits older than 30 calendar days (by `manifest.committed_at`) are **not** kept on the branch |
| Correctness | Same as today: pack self-check before push; verify fetch + decrypt after push; no local generation bump on failure |

Expected branch length: **at most ~30 commits** per machine branch under steady daily use (fewer if the appliance was idle for gaps).

### Recommended mechanism: coalesce + parent skip

Use manifest `committed_at` (already set in `Pack`) as the logical day key, with git commit parent pointers to shape history.

**1. Resolve current tip** (existing ref GET + commit/m manifest parse).

**2. Choose parent for the new commit**

- **Same UTC day as tip manifest:** parent = tip’s parent (drop today’s previous commit SHA from the branch line; replace today).
- **New UTC day:** parent = tip (append).
- **After choosing parent, prune depth:** walk ancestors from the chosen parent. While the chain has more than 29 commits *or* the oldest ancestor’s `committed_at` is before `now - 30 days`, move the parent to *its* parent (skip older commits). The new commit is then attached above the oldest retained daily commit.

Net effect: linear history, one commit per day in the window, tip always newest generation.

**3. Build tree and push** (blobs + tree; prefer full three-file tree without relying on stale `base_tree` for replaced days).

**4. Verify and update spec** (unchanged `sync.go` contract).

**Empty branch:** unchanged (create ref on first commit).

### What “rollback” means for operators

Within 30 days, `git log` on the branch shows one snapshot per day. Import can target an older SHA (admin/CLI) if product exposes it later; v1 minimum is tip import. Older than 30 days is not on the branch (not promised on GitHub).

### Stress points and mitigations

| Risk | Mitigation |
| --- | --- |
| Parent walk needs N GitHub API reads when pruning | Cap walk at ~32 commits; prune runs only on push (debounced), not on every save burst |
| Clock / UTC boundary | Use UTC for day bucketing; document in runbook |
| Failed verify after ref moved | Do not advance local generation; tip on GitHub may be ahead—retry push or import tip |
| Manifest `committed_at` missing on old commits | Migration: treat git commit author timestamp as fallback for pruning only |
| Replacing today leaves dangling tip commit | Unreachable from branch; acceptable until GC (same as orphan) |
| Depth skip mistakes could drop a day inside the window | Tests: same-day replace, day rollover, day-31 prune, verify still passes |

### Alternatives considered

- **Orphan-only tip (no history)** — Rejected by author: need 30-day daily history plus latest.
- **Unlimited linear history** — Rejected: unbounded repo growth.
- **Count-based cap without daily coalesce** — Rejects operator “1 per day” rule.
- **Scheduled squash job** — Extra scheduler and failure modes; coalesce-on-push is enough.

### Ideal outcome

Repo growth per branch is bounded (~30 commits × fleet branches). Operators keep recent daily restore points on GitHub. The tip stays correct and verified after every push.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  - Intraday commit replacement (one commit per UTC day)
  - Rolling 30-day retention on each blueprint upstream branch
  - Preserve pack self-check and post-push verify before updating sync metadata
  - Per-branch behavior; do not prune other machines’ branches or `main`
  - Default-on for new and upgraded controllers

* **Out-of-Scope Non-Goals:**
  - Guaranteed immediate drop in GitHub “repository size” (GC timing)
  - Keeping more than 30 days without a product change
  - Cross-branch or whole-repo maintenance in one job
  - Changing generation or debounce semantics
  - Rebuilding old commits with new SHAs (full history rewrite); parent-skip only

## Alternatives Weighed

- **Rejected: orphan tip every push** — No retained history; contradicts 30-day requirement.
- **Rejected: separate repo per machine** — Conflicts with single-repo fleet model.
- **Rejected: blob dedup only** — Does not cap commit count or intraday noise.

## Key Decisions & Staging for Next Steps

1. **`GitHub.Push` (or sync layer)** — Parent selection: same-day replace, new-day append, 30-day parent skip using manifest dates.
2. **`Github` helpers** — Resolve parent SHA chain; read manifest from commit tree (reuse `fetchBundleAtCommit` for manifest only where needed).
3. **Tests** — Same-day second push uses grandparent; day-31 push drops oldest day; empty branch; verify contract unchanged.
4. **Runbook** — Document UTC daily snapshots, 30-day retention, tip is canonical; optional import from older SHA within window.
5. **Constants** — `blueprintRetentionDays = 30` in code; optional future `blueprint.github.retention_days` if operators need tuning.

Handoff: `/egm-draft` on this file when you want a full design and implementation checklist.