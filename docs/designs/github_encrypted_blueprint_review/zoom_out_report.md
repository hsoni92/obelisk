# Goldfish Review Report — Stage 5 Initial Zoom Out

**Target:** `docs/designs/github_encrypted_blueprint.md` (full file)

## Reviewer Status

| Reviewer | Status |
| --- | --- |
| egm-zoom-out | completed |
| egm-inventive-catalyst | completed |

## Prioritized Findings

### OAuth complexity vs brief — Major
* **Lens:** zoom-out, inventive
* **Scope:** D1, Implementation Plan
* **Discrepancy:** Concepts brief emphasized deploy-key pairing; D1 adds full OAuth stack.
* **Recommendation:** Document why OAuth is required vs deploy key, or scope OAuth to admin login only.

### Dual control-plane with obelisk.md — Blocker
* **Lens:** inventive
* **Scope:** D3 vs obelisk.md CUJ 5 `pull_config`
* **Discrepancy:** Parent doc still S3-config-centric; no unified `pull_control` routing.
* **Recommendation:** Abstract CUJ 5; update obelisk.md in same change train.

### GA gated on stub DR — Blocker
* **Lens:** inventive, zoom-out
* **Scope:** D4, P4b
* **Discrepancy:** `RunNow()` stub; D4 requires full chain for GA.
* **Recommendation:** Capability matrix + E2E fixture plan; `phase_not_implemented` until P4b done.

### config.Save hook — Major
* **Lens:** zoom-out
* **Scope:** Implementation Plan
* **Recommendation:** Central hook in `config.Save`, not N adminapi wrappers.

### S3 config migration missing — Major
* **Lens:** zoom-out, structural
* **Recommendation:** Migration subsection for mirror-on vs legacy buckets.

### Session unlock vs OAuth refresh — Blocker
* **Lens:** inventive
* **Recommendation:** Segment `transport_auth` vs `content_auth`; `sync_status: locked` on restart.

### Six-package premature split — Major
* **Lens:** zoom-out
* **Recommendation:** Consolidate `internal/blueprint` for v1.

### Stale OQ3 in Alternatives — Nit
* **Scope:** ~L218
* **Recommendation:** Reference D3.