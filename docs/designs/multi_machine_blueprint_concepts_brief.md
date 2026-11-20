# Multi-machine blueprint (single repo, branch-per-machine)

* **Author:** hsoni92
* **Date:** 2026-11-20

## Overview

One GitHub login can run several Obelisk appliances without destroying each other's encrypted blueprint bundles. The fleet shares one private repository (default `login/obelisk-appliance`). Each appliance owns a dedicated git branch as its push upstream (default `machine-<machine.id>`). Optional `blueprint.follows[]` entries pull settings from another machine's branch on demand; they never change where this appliance pushes. OAuth stays one login per appliance; no extra repos are created per machine.

## Analysis

### The problem

Today every appliance defaults to the same repository and branch (`main`). A second machine with generation 0 can overwrite the first machine's bundle because `localNeverSynced` skips the remote-ahead check. Status and onboarding logic probe `main`, so a new host is steered toward another machine's bundle before it has its own branch. The operator wanted one shared repo, not N repositories.

### Authority and topology

| Source | Role | Writes to GitHub? | Wins on conflict |
| --- | --- | --- | --- |
| Local disk during operation | Live desired state | Only via push to own branch | Local until operator imports or follow-pull |
| Own `blueprint.github` (repo + branch) | This machine's upstream | Yes, push-only debounced save | Generation + content hash on that branch only |
| Each `follows[]` entry | Read-only mirror source | Never | Follow pull overwrites only configured fields; own branch and `machine.id` stay local |

`main` may hold a human README only. Machine state must not default there after this change.

### Branch naming

- Default ref: `machine-` + full `machine.id` (e.g. `machine-obelisk-a1b2c3d4e5f6`). No slashes in branch names (GitHub refs API).
- Set at first onboard from `machine.id`; not editable in onboarding UI afterward. DR adopt/import may change `machine.id` but does not repoint upstream branch as a casual rename.
- Shared repo name stays `obelisk-appliance` (or operator override); unchanged from today.

### Clobber guard and generation

Generation is monotonic per upstream branch, not per repository. Defense in depth when two appliances target the same branch: refuse push (including force push) when the remote manifest's `machine_id` differs from the local `machine.id` and the local appliance has never successfully pushed, or when the operator has not chosen explicit adopt/DR recovery. Reuse existing import/adopt vocabulary in errors.

Primary isolation is distinct branches, not the guard alone.

### `follows[]` semantics (v1)

- Pull only, triggered by operator (`Sync now` / CLI). No silent background pull (`AutoFollow` deferred).
- Fleet passphrases match: the operator uses the **same** blueprint passphrase as the source machine. Add-follow and pull fetch+decrypt must succeed with that passphrase; reject when source manifest `key_id` does not match the local unlock key (no cross-key follow in v1).
- **PullFollow** differs from Import: keep local `machine.id`, backup destination, and local `blueprint.github.repository` / `branch`; do not repoint upstream to the followed source.
- **Field scope:** settings from decrypted public spec; include `secrets.enc` only when `include_secrets: true`. Operator-facing copy must state that followed fields revert on the next pull.
- Follow metadata is local bookkeeping (like sync timestamps) and is excluded from `ContentHash` / canonical bundle bytes so follow edits do not spam pushes.

### Migration and rollout

**Automatic repoint on upgrade:** When an upgraded controller sees `blueprint.github.branch == main` (or empty normalized to `main`) and the remote bundle on `main` has `manifest.machine_id` equal to local `machine.id`, it runs a one-time migration: set local branch to `machine-<machine.id>`, push the current bundle to that new ref, and leave `main` unchanged (README or legacy commit only). No operator confirmation step for this happy path.

If `main` holds another machine's bundle, or `machine_id` mismatches, do not auto-migrate; use import/adopt wizard and manual branch configuration.

New onboard must set `branch = machine-<machine.id>` before remote probe or first push so discovery never treats another machine's `main` bundle as this host's home.

Brownfield: additive schema (`follows` optional); older controllers ignore unknown fields until upgraded.

### GitHub quota and status correctness

OAuth REST limits are shared across all appliances on one login. v1 must key remote probe cache by `(repository, branch)`, probe only the configured upstream branch (not `main` by default), and avoid polling storms on a fleet status page (longer interval or server-side cache than the single-panel 5s poll).

Structured operational logging is out of scope for this brief but should appear in the formal design as a major follow-up; today failures appear mostly as `last_push_error` strings.

### Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| Operator thinks repo = one machine | UI and runbook: "branch = this appliance"; show repo + branch on every sync message |
| Follow pull changes settings unexpectedly | Manual pull only; confirm scope; show last pull time per follow |
| Wrong passphrase on follow | Require same fleet passphrase; clear error when `key_id` differs |
| Two machines same branch | Branch default + machine_id clobber guard |
| Registry-free fleet discoverability | v1: operator names follows explicitly; no branch listing API required |

### Codebase touchpoints

`pkg/obelisk/blueprint.go` (schema, normalize defaults), `internal/blueprint/sync.go` (push guard), `internal/blueprint/runtime.go` (onboard, PullFollow, probe cache, attachRemote), `internal/blueprint/bundle.go` (canonical hash exclusions), `internal/blueprint/github.go` (unchanged fetch/push per branch), admin onboard UI and new Machines & Following surfaces, API routes for follow CRUD/pull, CLI parity.

Amends assumptions in `docs/designs/github_encrypted_blueprint.md` that treated one appliance on `main` as the whole story. Single-writer per branch remains; shared repo is allowed.

### Review tensions (Goldfish)

- **Scope:** Branch default + branch-aware discovery fixes clobber with small diff. `follows[]` is new surface area; keep v1 to manual pull and minimal UI.
- **YAGNI alternative:** Defer `follows[]` and use one-shot Import with branch override for profile copy. Rejected for this brief because the author wants ongoing follow with preserved own upstream in the same release train.
- **Inventive alternatives:** Registry file on `main`, path-per-machine on one branch, or per-branch deploy keys add isolation or UX but cost more code than branch-per-machine given existing branch plumbing. Not chosen for v1.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * One shared GitHub repo per login; default repo name unchanged.
  * Default upstream branch `machine-<machine.id>` for new onboard.
  * Machine_id-aware clobber guard on push (including force push) when branches collide.
  * `blueprint.follows[]` with add/remove, manual pull, optional `include_secrets`.
  * PullFollow preserving local identity and upstream.
  * Machines & Following UI; API and CLI parity for follow ops and extended status.
  * Probe cache and status keyed by repository + branch.
  * Automatic one-time migration from `main` to `machine-<machine.id>` on upgrade when manifest matches.

* **Out-of-Scope Non-Goals:**
  * Multiple repos per login (author constraint).
  * Folder-per-machine layout on `main` without branch isolation.
  * `AutoFollow` / scheduled pull (v2).
  * Branch listing or registry index on `main` (v2).
  * Multi-writer push to the same branch.
  * GitHub rulesets or per-machine deploy keys (v2 hardening).
  * Full structured logging/metrics implementation in the same slice (design doc should specify).

## Alternatives Weighed

| Alternative | Outcome |
| --- | --- |
| Repo per machine (`login/obelisk-<id>`) | Rejected by author: too many repos. |
| Branch per machine in one repo | **Chosen:** reuses existing `branch` in Fetch/Push/Import/ChainPull. |
| Folder per machine on `main` (`machines/<id>/…`) | Rejected for v1: needs path prefix support in GitHub file ops. |
| Branch names with slashes (`machine/<id>`) | Rejected: refs API path escape issues. |
| Defer `follows[]`; Import-only cross-branch copy | Rejected for this brief; noted as smaller v1 if scope must shrink. |
| Registry JSON on `main` mapping machines | Deferred: useful for DR discovery at scale, not required to stop clobber. |

## Key Decisions & Staging for Next Steps

1. **Locked:** Single repo; upstream branch defaults to `machine-` + full `machine.id`; fixed after first onboard (not UI-editable).
2. **Locked:** Auto-repoint on upgrade from matching `main` bundle to `machine-<machine.id>` with one-time push; `main` left as-is.
3. **Locked:** Clobber guard uses remote `machine_id` when generations allow overwrite today.
4. **Locked:** Follows are pull-only, manual in v1; PullFollow must not rewrite `blueprint.github` upstream fields.
5. **Locked:** Follow add/pull require the same passphrase / matching `key_id` as the source bundle; no cross-key follow in v1.
6. **Locked:** Follow sync fields excluded from content hash; follows treated like other local-only blueprint metadata.
7. **Staging:** Implement branch default + auto-migration + probe/discovery + clobber guard first; then schema + PullFollow + API/UI; integration test with two appliances, one login, two branches.

**CUJ seeds for draft:** (1) Second VPS onboard without touching machine A's branch. (2) Push refused when misconfigured onto foreign branch with adopt message. (3) Add follow, manual pull, local upstream unchanged. (4) Upgrade existing `main` install to dedicated branch.