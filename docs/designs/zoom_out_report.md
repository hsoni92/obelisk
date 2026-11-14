# Goldfish Review Report — Stage 5 Initial Zoom Out

**Target:** `docs/designs/mail_gateway.md` (full file)

## Reviewer Status

| Reviewer | Status |
| --- | --- |
| egm-zoom-out | completed |
| egm-inventive-catalyst | completed |

## Verdict

**Does not pass** zoom-out bar for v1.0 as written — fit on appliance is strong; trim attachments, full Artemis port, IMAP-on-add, and resolve shared k3s credentials blocker before M1.

## Prioritized Findings

### k3s credentials write path — Blocker

* **Reviewer Lens:** egm-zoom-out
* **Discrepancy:** Mail duplicates OpenRouter sidecar writes while OQ1 admits read-only controller mount.
* **Recommendation:** Fix once for OpenRouter + mail (Secret sync or writable subpath) before M1.

### Full Artemis host map port — Major

* **Reviewer Lens:** egm-zoom-out
* **Discrepancy:** G6 mandates full parity tests; brief caps at ~10 operator mailboxes.
* **Recommendation:** v1.0 SMTP presets for 3–5 domains; expand with v1.1 read.

### Attachments 10 MB in v1.0 — Major

* **Reviewer Lens:** egm-zoom-out
* **Discrepancy:** CUJs are transactional text send; MIME attachment path is large scope.
* **Recommendation:** v1.0 send = text/html only; defer attachments.

### IMAP validation on add vs send-first — Major

* **Reviewer Lens:** egm-zoom-out
* **Discrepancy:** G2 requires IMAP on add; M0–M3 is send-only.
* **Recommendation:** v1.0 add = SMTP AUTH only.

### Failover vs exactly-once — Major

* **Reviewer Lens:** egm-inventive-catalyst
* **Discrepancy:** OQ3 duplicate-send left open.
* **Recommendation:** Optional `Idempotency-Key` + TTL dedup before cross-account retry.

### Parallel gateway without shared abstraction — Major

* **Reviewer Lens:** egm-zoom-out
* **Recommendation:** Extract shared credentials/gateway shell before third exposure feature.

### v1.1 files in v1.0 manifest — Major

* **Reviewer Lens:** egm-zoom-out
* **Recommendation:** Gate `imap_read.go` / `read.go` behind explicit v1.1 milestone.

### GET /accounts data plane — Nit

* **Reviewer Lens:** egm-zoom-out
* **Recommendation:** Defer or restrict to admin-only.