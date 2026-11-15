# Stage 7 — Critic Report (Feature spec, lines 1–231)

## Reviewer Status

| Reviewer | Status |
| --- | --- |
| egm-design | completed |
| egm-structural | completed |
| egm-style | completed |

## Blockers

### Missing Critical User Journeys
* **Lens:** structural, design
* **Scope:** Between Non-Goals and Technical Plan
* **Recommendation:** Add 6–8 CUJs (enable sync, OAuth device, unlock TTL, chained restore, key loss, remote-ahead, G6 S3-only path).

### OAuth vs machine credential in Problem (L22)
* **Lens:** design, style, comprehension
* **Recommendation:** Align with D1; fix mermaid “Pair credential” node.

### Incomplete Risks & Mitigations (feature-spec standard)
* **Lens:** structural, style
* **Recommendation:** Top-level section: key loss, token revoke, GitHub outage, locked session, generation conflict.

## Major

* G6 coexistence: when mirror **off**, S3 `config/` behavior must be explicit.
* Alternatives: stale OQ3 (L218); “Rejected by author” preference prose.
* No narrative cross-link to `obelisk.md` CUJ 5 rewrite.
* Resolved decisions placement vs EGM template (CUJs missing).
* Problem bullet 3 still lists four stores without D3 winner.