# Obelisk Mail Gateway

* **Author:** hsoni92
* **Date:** 2026-11-14
* **Status:** Ideation (brainstorm)

--------------------------------------------------------------------------------

## Overview

Obelisk operators run a single-node middleware appliance with an existing **OpenRouter gateway** pattern: many upstream accounts, one gateway bearer token, Kong public path, admin CRUD under **Exposure**. This feature adds a **Mail gateway** on the same model.

Apps anywhere (Vercel, same host, LAN) call **`{origin}/mail/api/v1`** with the gateway token to **send** mail through a pool of operator-owned mailboxes (consumer SMTP with app passwords, Artemis-style host presets). **Read** (IMAP list/get per account) ships in a **second delivery stage** on the same token and routes so the operator gets send quickly without blocking on a full mail client in Go.

Obelisk speaks HTTP only at the edge; SMTP and IMAP run **only on the controller**. This is programmatic mail for indie projects (transactional volume, OTP-style reads), not hosted email for humans and not bulk marketing.

## Analysis

### Gateway model (locked for v1)

- **Data plane:** separate controller listen port (e.g. `:8082`), Kong route `/mail` with `strip_path: true`, public paths under `/mail/api/v1`.
- **Auth:** one gateway bearer token (same ergonomics as OpenRouter); constant-time compare; token is a **high-privilege appliance secret** (send + later read on all pool mailboxes).
- **Control plane:** `mail_gateway` in `obelisk.yaml`, secrets in `mail.credentials.json` beside config (mode `0600`), admin API `/api/v1/mail/*`, Exposure panel cloned from OpenRouter.
- **Consumers:** ordinary HTTPS clients; no Vercel-specific code. IMAP never runs on serverless.

### Mailbox pool and routing

| Operation | Rule |
| --- | --- |
| **Send** (no `account_id`) | Round-robin with failover; response must return `account_id` and `from`. |
| **Send** (with `account_id`) | That mailbox only unless explicit `failover: true`. |
| **Read** | Always requires `account_id`; no merged pool inbox. |

Rotation cursor should persist across restarts (file under `/var/lib/obelisk/` on appliance), not copy OpenRouter’s snapshot-only iteration blindly.

### Credentials split

Account metadata (id, label, address, enabled, provider preset) in YAML; SMTP/IMAP secrets in JSON map keyed by account id. Admin add validates live (SMTP AUTH + IMAP login where supported). Duplicate addresses rejected. Unsupported domains (Proton bridge-only, Tutanota no-IMAP) rejected at add time like Artemis.

### Provider host maps

Port Artemis `smtp-hosts.ts` and `hosts.ts` into Go (or generated parity tests vs TS). App-password-required domains (Gmail/Yahoo/AOL) enforced on add. Outlook STARTTLS vs Gmail implicit TLS must match Artemis.

### Exposure and abuse

Kong **rate-limiting** and **request-size-limiting** on `/mail` (stricter than OpenRouter). Mail bombing and token brute force are realistic; document that the route must not be treated as a public anonymous API. Funnel on mail routes is **out of scope for v1** (default deny).

### Operational limits

- **Scale:** operator-scale — roughly ≤10 mailboxes, transactional sends spread across pool; not linear “N accounts = N× bulk throughput.”
- **Attachments:** pass-through up to ~10 MB at gateway; list metadata-only; full body/attachments on get with caps.
- **Deliverability:** operator owns DNS/reputation per mailbox; appliance does not guarantee inbox placement.

### Inventive resolution (send vs read vs Vercel)

| Tension | Resolution |
| --- | --- |
| Serverless cannot hold IMAP | Read is HTTP on Obelisk; workers connect IMAP on controller. |
| User wants instant send+read | **Stage:** send gateway first, read gateway second; one token throughout. |
| Consumer quotas | Health-aware pool: skip cooled-down accounts; surface `retry_after`. |
| OTP / wait patterns | **v1.1+:** optional `wait-for-mail` (correlation id + timeout), not blocking IMAP inside HTTP handler. |
| Large send volume | Optional queued send (accept 202 + job id) if sync SMTP exceeds HTTP timeouts — defer unless draft proves need. |

### Goldfish reviewer status

| Lens | Status |
| --- | --- |
| egm-design | completed |
| egm-zoom-out | completed |
| egm-codebase-details | completed |
| egm-comprehension | completed |
| egm-implementation-readiness | completed |
| egm-structural | completed |
| egm-inventive-catalyst | completed (summary + phased IFR) |

### Glossary

| Term | Meaning |
| --- | --- |
| **Mail gateway** | HTTP API on controller; not SMTP port 25 on the internet. |
| **Mailbox / account** | One operator-added identity (`account_id` + email address). |
| **Pool** | Set of enabled accounts used for send rotation. |
| **Gateway token** | Bearer secret for HTTP clients; not per-mailbox password. |
| **Send accepted** | SMTP upstream accepted the message; delivery/bounce is provider-dependent. |

## Goals vs. Non-Goals

### In-Scope Goals (v1 — send stage)

* **G1 — Exposure feature:** Enable/disable mail gateway; rotate gateway token; Kong path `{origin}/mail/api/v1`.
* **G2 — Mailbox pool:** Admin add/remove N accounts with validation and duplicate rejection.
* **G3 — HTTP send:** `POST /mail/api/v1/send` with optional `account_id`, attachments ≤10 MB, JSON errors safe for UI.
* **G4 — Pool routing:** Round-robin + failover; response includes `account_id` and `from`.
* **G5 — Consumer access:** Any HTTPS client with token (including Vercel env bearer).
* **G6 — Artemis parity (SMTP):** Domain→host map and app-password policy for supported providers.
* **G7 — Operator test:** Admin “test send” from UI without reading logs.

### In-Scope Goals (v1.1 — read stage, same product line)

* **G8 — HTTP read:** `GET .../accounts/{id}/messages` (metadata), `GET .../messages/{uid}` (body).
* **G9 — IMAP on controller only:** Connect-per-request or small pool; timeouts; secret redaction in errors.
* **G10 — Optional:** `wait-for-mail` helper for OTP flows (correlation + timeout).

### Out-of-Scope Non-Goals (v1)

* Raw SMTP/IMAP TCP proxy through Kong.
* Artemis per-user mailbox nodes or vault coupling.
* Shared blurred inbox across accounts.
* Persistent mail store (Postgres/S3 archive) in v1.
* Webhooks / IMAP IDLE push in v1.
* OAuth provider login in v1 (app passwords only).
* Bulk marketing, list management, warm-up.
* Full webmail UI; SPF/DKIM management on appliance.
* Multi-tenant per-app tokens in v1 (one gateway token).
* Proton Bridge / Tutanota as pool members.
* Public Funnel exposure for mail routes.

## Alternatives Weighed

| Alternative | Decision |
| --- | --- |
| **Third-party API only (Resend/SendGrid)** | Rejected by author: wants self-hosted pool on the appliance using existing mailboxes. |
| **Separate mail microservice** | Rejected by author: duplicates ops; controller gateway matches OpenRouter. |
| **Raw SMTP/IMAP proxy** | Rejected by author: HTTP-only consumer surface and Kong auth/rate limits. |
| **Shared inbox read API** | Rejected by author: read must be per `account_id`. |
| **Ship send+read in one first release** | Rejected after review: read is a second-stage delivery to avoid a multi-month v1. |
| **Node sidecar reusing Artemis imapflow** | Rejected by author: Obelisk controller is Go; port behavior, do not split runtime. |
| **Persist all messages in RustFS** | Rejected for v1: stateless proxy; storage adds retention/compliance scope. |
| **Copy OpenRouter snapshot order as “round-robin”** | Rejected: mail needs real rotation cursor and explicit `from` semantics. |

## Key Decisions & Staging for Next Steps

### Locked decisions (author, 2026-11-14)

| ID | Decision |
| --- | --- |
| M1 | YAML key `mail_gateway`; credentials `mail.credentials.json`; package `internal/mailgateway`. |
| M2 | Public path `/mail/api/v1`; listener `:8082` (parallel to OpenRouter `:8081`). |
| M3 | One gateway bearer token in v1; document blast radius. |
| M4 | Send stage ships before read stage; same token and admin panel. |
| M5 | Provider presets from Artemis with Go parity tests. |
| M6 | Kong rate limit + body size limit on `/mail`. |

### Left for `/egm-draft`

* Exact JSON schemas for send/list/get and error catalog.
* Go libraries (`net/smtp` vs `go-mail`; IMAP via `emersion/go-imap` or similar).
* Message id scheme (`uid` vs Message-ID header).
* K8s writable path for credentials (OpenRouter shares read-only mount risk).
* Duplicate-send policy on SMTP failover retry.
* Structured logging/metrics per send (account_id, outcome, latency).
* `wait-for-mail` contract and max wait time.

### Proposed staging

| Stage | Delivers |
| --- | --- |
| **M0** | Spec + schema + admin CRUD + credentials file (no public send) |
| **M1** | Send handler + selector + bearer auth + direct port |
| **M2** | Kong route + runbook + rate limits |
| **M3** | Admin Exposure panel + test send |
| **M4** | Read list/get per account (v1.1) |
| **M5** | `wait-for-mail` optional (v1.1+) |

### Critical user journeys (for draft)

1. **Enable gateway** — operator enables mail, adds mailbox, rotates token, copies base URL.
2. **Vercel send** — app POSTs send with bearer; receives `account_id` + `from`.
3. **Pool failover** — first mailbox rate-limited; gateway tries next; client sees actionable error if all fail.
4. **Read OTP (v1.1)** — app GETs messages for fixed `account_id`; filters client-side or via query.
5. **Token rotation** — old bearer returns 401; operator updates app env.

--------------------------------------------------------------------------------

*Brainstorm input: OpenRouter gateway in obelisk; Artemis reference at `artemis/ui/lib/notifications/*` and `artemis/ui/lib/mail/*`.*