# Consolidated Goldfish Review — Mail Gateway

**Target:** [`mail_gateway.md`](./mail_gateway.md)  
**Date:** 2026-11-14  
**Mode:** Consolidated (Stages 5–9)

## Reviewer status

| Stage | Report | Status |
| --- | --- | --- |
| 5 Initial zoom out | `zoom_out_report.md` | completed |
| 6 Comprehension | `comprehension_report.md` | completed |
| 7 Critic | `critic_report.md` | completed |
| 8 Intermediate zoom out | `intermediate_zoom_out_report.md` | completed |
| 9 Implementation readiness | `implementation_readiness_report.md` | completed |

## Overall verdict

**Not approved for M0/M1 implementation.** Architecture (OpenRouter-style Exposure gateway on controller) is sound. The spec must close operational blockers and trim v1.0 scope before coding.

## Traceability — top actions

| ID | Sev | Source | Action |
| --- | --- | --- | --- |
| A1 | 🔴 | 5,7,8,9 | Resolve writable credentials on k3s (shared with OpenRouter) |
| A2 | 🔴 | 7,8 | Add NFR + Risks & Mitigations; close OQ3 (failover/idempotency) |
| A3 | 🔴 | 9 | Freeze admin API + RTK JSON contract before UI |
| A4 | 🟡 | 5,8 | v1.0 send: text/html only; SMTP-only validation on add; 3–5 presets |
| A5 | 🟡 | 6,8 | Glossary: Artemis, Exposure, `{origin}`, platform-ready |
| A6 | 🟡 | 9 | Helm volume for `/var/lib/obelisk` cursor; `go.mod` deps |
| A7 | 🟡 | 9 | Reconcile Kong dbless dev/helm; set numeric rate limits |
| A8 | 🟢 | 5,8 | Gate v1.1 files out of v1.0 manifest; defer GET /accounts |

## Suggested doc edit order

1. Add **Non-Functional Requirements** and **Risks & Mitigations** sections.
2. Close **OQ1** (credentials) and **OQ3** (no cross-account retry without `Idempotency-Key`, or fail closed).
3. Trim **G2/G3/G6** for v1.0 scope per zoom-out.
4. Add **Glossary** and admin API contract appendix for Stage 9.
5. Re-run `/egm-review` on updated `mail_gateway.md` or use `/egm-resolve` on findings.

## Related artifacts

- Brainstorm: [`email_gateway_concepts_brief.md`](./email_gateway_concepts_brief.md)