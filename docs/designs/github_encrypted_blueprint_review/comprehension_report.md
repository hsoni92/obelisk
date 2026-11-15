# Stage 6 — Comprehension Report

**Target:** `github_encrypted_blueprint.md` (FeatureRange 1–231, DetailedRange 232–end)

## What this proposes

Opt-in encrypted push-only GitHub mirror of control-plane state; S3 `data/` only when mirror on; chained restore with passphrase unlock.

## How the existing system works, per this document

Local `obelisk.yaml` is authoritative; DR uses S3 under `machine.id`; gap is off-site encrypted settings copy and orchestrated rebirth.

## Findings summary

| Severity | Topic |
| --- | --- |
| Blocker | OAuth vs deploy-key wording in Problem (L22) |
| Blocker | Funnel undefined in chain |
| Blocker | `machine.id` undefined |
| Major | Sidecars/instances undefined for readers |
| Major | `generation` semantics underspecified |
| Major | Stale OQ3 reference |
| Nit | CUJ acronym, Recovery Kit format |