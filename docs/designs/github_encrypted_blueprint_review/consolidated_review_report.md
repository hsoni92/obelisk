# Consolidated Goldfish Review Report

**Target:** [github_encrypted_blueprint.md](../github_encrypted_blueprint.md)  
**Mode:** Uninterrupted full flow (Stages 5–9)  
**Date:** 2026-11-14  
**FeatureRange:** 1–231 | **DetailedRange:** 232–360

## Executive summary

The **encrypted blueprint mirror** direction matches the DR gap and your resolved decisions (OAuth, split bundle, no S3 `config/` fallback, full chain for GA). The spec is **not approval-ready**: it lacks **Critical User Journeys**, has **auth wording contradictions** (Problem vs D1), is **out of sync with [obelisk.md](../obelisk.md) CUJ 5**, and the **Implementation Plan** fails the zero-decision rule (wrong paths, undecided crypto/OAuth/orchestrator details). Expect a **large release train** (stub backup/restore today) before GA is honest.

## Reviewer status

| Stage | Reviewers | Artifact |
| --- | --- | --- |
| 5 | zoom-out, inventive-catalyst | [zoom_out_report.md](./zoom_out_report.md) |
| 6 | comprehension | [comprehension_report.md](./comprehension_report.md) |
| 7 | design, structural, style | [critic_report.md](./critic_report.md) |
| 8 | zoom-out | [intermediate_zoom_out_report.md](./intermediate_zoom_out_report.md) |
| 9 | impl-readiness, codebase-details, style | [implementation_readiness_report.md](./implementation_readiness_report.md) |

## Prioritized action items

### Blockers (fix before `/egm-review` pass 2 or implementation)

| ID | Finding | Sources | Recommended fix |
| --- | --- | --- | --- |
| B1 | No `## Critical User Journeys` | structural, design, zoom-out | Add CUJs: enable+wizard, device OAuth, day-2 push/unlock, chained restore, key loss, remote-ahead import, mirror-off S3 path |
| B2 | Problem L22 still says deploy key/GitHub App | style, comprehension, inventive | Rewrite to D1 OAuth; fix flowchart “Pair credential” |
| B3 | obelisk.md CUJ 5 vs `pull_blueprint` | inventive, structural, impl-ready | Add `pull_control` routing narrative; draft CUJ 5 dual-path; link parent doc |
| B4 | Transport (OAuth) vs content (passphrase) conflated | inventive, comprehension | Subsection: two lifecycles; `sync_status` when locked or after pod restart |
| B5 | Implementation plan not executable | impl-ready, codebase-details | Central `config.Save` hook + suppress list; fix file paths; crypto manifest schema; orchestrator persistence |
| B6 | GA (D4) vs stub backup/restore | inventive, zoom-out | Capability flags; E2E with MinIO fixtures; no UI “DR ready” until P4b green |

### Major

| ID | Finding | Recommended fix |
| --- | --- | --- |
| M1 | Stale “OQ3” in Alternatives (~L218) | Point to D3; migration playbook |
| M2 | No top-level **Risks & Mitigations** | Lift security notes + G5 into dedicated section |
| M3 | G6: mirror-off appliances — S3 `config/` | Explicit row in authority table |
| M4 | Undefined terms: `machine.id`, sidecars, Funnel, generation | Glossary or short definitions in Overview |
| M5 | Alternatives “Rejected by author” tone | Objective trade-off prose |
| M6 | Phase table / verification not wired | Phase → tests → GA gate matrix |
| M7 | Credential path asymmetry (Helm RO `/etc/obelisk`) | Document bundle pack paths + `OBELISK_CREDENTIALS_DIR` |
| M8 | OAuth justification record | Why OAuth over deploy key given headless push (author decision D1) |

### Nits

* Terminology: standardize “encrypted blueprint mirror”  
* Recovery Kit format optional detail  
* `oauth_login` in yaml — low DR value  
* key_id rotation UX defer post-GA  

## Traceability matrix (sample)

| Consolidated | Stage reviewers | Section |
| --- | --- | --- |
| B1 Missing CUJs | structural, zoom-out 8 | Template |
| B2 Auth contradiction | style, comprehension, inventive | L22, D1 |
| B5 Save hook | codebase-details, impl-ready, zoom-out | Implementation |
| B3 CUJ 5 drift | inventive, structural | obelisk.md |
| M1 OQ3 stale | structural, style, comprehension | Alternatives |
| B6 Stub DR | codebase-details, inventive | D4, P4b |

## Suggested remediation order

1. Editorial pass on feature spec (B1–B4, M1–M5, glossary).  
2. Update `obelisk.md` CUJ 5 + cross-links (B3).  
3. Rewrite Implementation Plan: paths, single hook, crypto/OAuth/orchestrator specs (B5).  
4. Re-run **Implementation plan only** or full review.  

## Author alignment note

You locked **OAuth (D1)** and **full chain GA (D4)**. Reviewers still flag **complexity** vs deploy-key minimalism; that is a trade-off you accepted. The review does **not** ask to revert OAuth—it asks to **document** it consistently and **segment** transport vs encryption sessions.

---

*End of consolidated report.*