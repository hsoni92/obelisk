# OpenRouter Gateway

* **Author:** hsoni92
* **Status:** Draft
* **Date:** 2026-11-14

--------------------------------------------------------------------------------

## Overview

Obelisk operators often run several **OpenRouter** provisioning accounts (separate sign-ups, separate credit buckets). Client apps expect a **single OpenRouter-compatible base URL** and API shape ([OpenRouter quickstart](https://openrouter.ai/docs/quickstart)): `https://openrouter.ai/api/v1/...`, `Authorization: Bearer <key>`, optional `HTTP-Referer` / `X-OpenRouter-Title`.

**OpenRouter Gateway** is an Obelisk platform component that:

1. Exposes a **native-compatible HTTP API** on the appliance (via Kong).
2. **Round-robins** upstream calls across **N configured OpenRouter accounts** (only routing policy in v1—no model-based routing, no weighted LB).
3. Lets the operator manage accounts in the **admin UI** (add, disable, label, rotate keys).

Clients point `baseURL` at the gateway (e.g. `https://<vps>/openrouter/api/v1`) and use a **gateway-issued bearer token**. The gateway selects the next eligible upstream API key internally.

## The Problem

* Multiple OpenRouter accounts increase effective rate limits and cost isolation, but apps would otherwise need custom key rotation logic.
* Keys must not live in every client repo; the appliance should hold upstream secrets and present one stable endpoint.
* OpenAI-SDK compatibility (`baseURL` + `apiKey`) only works if the gateway mimics OpenRouter path layout and response shapes closely.

## Goals

* **G1 — API compatibility:** Proxy all OpenRouter REST paths under `/api/v1/*` (chat completions, legacy chat, embeddings, models, etc.) with **transparent** request/response bodies and **streaming** (SSE) pass-through unless OpenRouter returns an error before stream start.
* **G2 — Round-robin only:** Cyclic selection among **enabled** accounts; one account per request; no sticky sessions.
* **G3 — Account pool:** Operator configures **N accounts** (display label, upstream API key, enabled flag, optional per-account notes). Keys stored server-side; admin API masks secrets on read.
* **G4 — Failure handling:** On upstream **401/402/429** (and configurable 5xx), **retry once** on the **next** account in rotation before returning error to client (bounded retries, not infinite).
* **G5 — Admin UI:** New nav section **OpenRouter** (or under **Exposure**): list accounts, add/remove, enable/disable, test call (`GET /api/v1/models` via gateway), show last error per account (from gateway state).
* **G6 — Gateway auth:** Clients authenticate with a **single gateway API key** (generated in UI, stored hashed). Upstream OpenRouter keys never sent to clients.
* **G7 — Observability:** Log account id used per request (no prompt content in logs by default); counters in overview optional follow-on.

## Non-Goals

* **NG1:** Model routing, fallbacks, or cost optimization (OpenRouter’s own features stay on upstream; gateway does not reinterpret model choice).
* **NG2:** Caching responses or prompt logging in v1.
* **NG3:** Billing metering, quotas per client, or multi-tenant gateway keys (single operator appliance; one gateway key in v1).
* **NG4:** Replacing Kong for other HTTP; gateway is one routed service.
* **NG5:** EU-specific `eu.openrouter.ai` split unless operator sets upstream base URL per account (future).

## Open Questions

* **OQ1:** Persist account list in `obelisk.yaml` vs Kubernetes Secret only (yaml references secret names).
* **OQ2:** Gateway process: new Go binary in platform chart vs sidecar—preference Go service for parity with admin API.
* **OQ3:** Public exposure: Kong route on VPS only vs also Tailscale Funnel allowlist (default: same policy as admin API—not public Funnel).

## Critical User Journeys (CUJs)

### CUJ 1: Add two OpenRouter accounts

* **Given:** Platform running; operator has two OpenRouter API keys.
* **When:** Operator opens **OpenRouter** in admin UI, adds Account A and B, saves, copies gateway base URL + gateway bearer token.
* **Then:** Both accounts show enabled; test button returns models JSON; secrets masked in UI.

### CUJ 2: Client drop-in (OpenAI SDK)

* **Given:** Gateway URL and gateway token configured.
* **When:** Client sets `baseURL` to `https://<host>/openrouter/api/v1` and `apiKey` to gateway token; calls `chat.completions` or OpenRouter `responses` path per their SDK.
* **Then:** Response matches direct OpenRouter behavior; alternating requests use A/B/A/B upstream keys (verified in gateway metrics/logs).

### CUJ 3: Upstream rate limit

* **Given:** Account A returns HTTP 429.
* **When:** Client sends a completion request.
* **Then:** Gateway retries with Account B; client succeeds if B healthy; if all accounts fail, client receives upstream-style error body/status.

## Technical Plan (Architecture)

```text
Client (OpenAI SDK / curl)
    │  Authorization: Bearer <GATEWAY_KEY>
    ▼
Kong (path /openrouter/api/v1/*)
    ▼
openrouter-gateway (Go, platform pod)
    │  pick next enabled account (atomic RR index)
    │  Authorization: Bearer <UPSTREAM_KEY_N>
    ▼
https://openrouter.ai/api/v1/*  (or per-account base override)
```

**State:**

* `spec.openrouter_gateway` in `obelisk.yaml`: enabled, gateway token hash, accounts[] `{ id, label, enabled, secret_ref }`.
* Kubernetes Secret `obelisk-openrouter-upstream` holds upstream keys keyed by account id.
* Optional Secret `obelisk-openrouter-gateway` for generated gateway token.

**Round-robin:**

* Mutex-protected `nextIndex % len(enabledAccounts)`.
* Skip disabled accounts when advancing.

**Compatibility:**

* Reverse proxy: preserve method, path suffix after strip prefix, query string, body.
* Forward client headers except `Authorization` (replaced with upstream key).
* Pass through `HTTP-Referer`, `X-OpenRouter-Title` from client or gateway defaults from spec.
* Stream: `io.Copy` with flush on upstream `text/event-stream`.

**Admin API (sketch):**

* `GET/PATCH /api/v1/openrouter/config`
* `POST /api/v1/openrouter/accounts` | `PATCH/DELETE .../accounts/{id}`
* `POST /api/v1/openrouter/test`

**Exposure:** Kong route `obelisk-openrouter-gateway:8081` → service; not on RustFS Funnel allowlist by default.

## Alternatives Considered

### Alternative A: Client-side key rotation

* **Pros:** No new service.
* **Cons:** Secrets in every app; no central observability.
* **Rejected by author:** Defeats appliance control plane.

### Alternative B: Use OpenRouter’s built-in fallbacks only

* **Pros:** Less code.
* **Cons:** Does not merge **multiple billing accounts**; one upstream key per deployment.
* **Rejected by author:** User requirement is multi-account RR.

### Alternative C: Full API gateway product (APISIX, etc.)

* **Pros:** Features.
* **Cons:** Operational weight on small VPS; still need custom RR across auth headers.
* **Rejected by author:** Scope.

## Non-Functional Requirements

* **Security:** Upstream keys and gateway token at rest in Secrets; TLS termination at Kong; no prompt logging in v1.
* **Latency:** One extra hop on appliance; target &lt;50ms overhead excluding upstream.
* **Failure:** If zero enabled accounts, `503` with JSON error explaining misconfiguration.

--------------------------------------------------------------------------------

## Detailed Implementation Plan

### Backend (Go)

| File | Rationale |
|------|-----------|
| `pkg/obelisk/spec.go` | Add `OpenRouterGateway` struct and defaults in `DefaultSpec`. |
| `pkg/obelisk/openrouter.go` | Account types, validation, round-robin selector. |
| `internal/openroutergateway/proxy.go` | HTTP reverse proxy, streaming, retry on 401/402/429. |
| `internal/openroutergateway/server.go` | Gateway HTTP server mux matching `/api/v1/*`. |
| `internal/adminapi/openrouter.go` | CRUD handlers, mask secrets, test endpoint. |
| `internal/adminapi/server.go` | Register routes. |
| `internal/config/secrets.go` (or extend load) | Read/write openrouter secrets alongside backup creds pattern. |
| `cmd/openrouter-gateway/main.go` | Standalone entry for platform chart. |
| `deploy/helm/platform/templates/openrouter-gateway.yaml` | Deployment + Service. |
| `deploy/helm/platform/files/kong-dbless.yaml` | Route `/openrouter` prefix. |
| `deploy/kong/kong-dbless.yaml` | Dev parity route. |

### Admin UI

| File | Rationale |
|------|-----------|
| `web/admin/lib/types.ts` | OpenRouter account/gateway types. |
| `web/admin/lib/store/obelisk-api.ts` | RTK endpoints for openrouter config. |
| `web/admin/components/openrouter/openrouter-panel.tsx` | Account list, add form, gateway URL + copy token, test. |
| `web/admin/components/layout/app-shell.tsx` | Nav item. |
| `web/admin/app/page.tsx` | Section wiring. |

### Docs / schema

| File | Rationale |
|------|-----------|
| `schemas/obelisk.yaml.schema.json` | `openrouter_gateway` section. |
| `docs/runbooks/openrouter-gateway.md` | Operator setup, SDK example. |
| `docs/designs/SITEMAP.md` | Register design link. |

### Tests

| File | Rationale |
|------|-----------|
| `pkg/obelisk/openrouter_test.go` | RR selector, skip disabled. |
| `internal/openroutergateway/proxy_test.go` | httptest upstream mock, retry behavior. |