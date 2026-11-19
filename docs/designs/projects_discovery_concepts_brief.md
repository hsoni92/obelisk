# Projects (connection discovery groups)

* **Author:** hsoni92
* **Date:** 2026-11-19

## Overview

Remote apps (Vercel, Railway, a laptop backend) need Obelisk connection strings without signing into the admin UI with GitHub. Today the operator copies URLs per instance from **Services** or calls `GET /api/v1/connections` behind a GitHub session.

This feature adds **Projects** in the Services workflow: named groups that reference enabled service instances. The same instance may appear in more than one Project. Each Project has a **GET-only discovery URL** protected by a **revocable bearer token**. A successful call returns a JSON catalog of **full connection URLs with credentials** for every instance in that Project—the same substance as the admin connections view, filtered by membership.

Membership is **control-plane only**. Nothing self-registers. When the operator adds or removes an instance in the UI, the next discovery response reflects it.

The dormant `projects[]` field in `obelisk.yaml` today (`name`, `postgres_db`, `redis_db`) was never wired to the UI or to `BuildConnections`. This feature **redefines Projects** as discovery groups. The old fields are retired in favor of stable `id`, display `name`, and `instance_ids[]`.

## Analysis

**Schema collision (resolved).** The previous `Project` model meant “Postgres DB name + Redis index on shared instances.” That was never implemented. The new Project is “which instances this app may discover.” Logical databases remain under **Services → instance → Logical databases**. A Project does not create a database unless we later add optional provisioning (out of v1 scope).

**Not generic service discovery.** Consul-style catalogs expose host, port, and health; clients bring credentials. Obelisk discovery returns ready-to-use `postgresql://`, `redis://`, and HTTP API URLs because the author chose convenience over a split response. Document this as an **authenticated connection export**, not a registry protocol. No `POST /register`, no heartbeats, no load balancing across replicas.

**Security tension (accepted with guardrails).** A bearer token that returns full URLs is a credential bundle. One leak exposes every grouped service. This is an intentional HTTP exception to “do not expose Postgres/Redis on the public internet”: access is **token-gated**, rate-limited, never on Tailscale Funnel, and revocable. Tokens live in **Kubernetes secrets** (or the credentials sidecar pattern used by mail/OpenRouter gateways)—**not** in git-tracked `obelisk.yaml` or the public blueprint. Discovery auth is **separate** from the admin `LocalToken` so a project token never grants admin API access.

**Auth placement.** Discovery must bypass GitHub session on its route only, using constant-time Bearer validation. Prefer `Authorization: Bearer`; supporting a token in the URL path or query is optional for copy-paste but logs and Referer risk—if offered, document the warning and redact in access logs.

**Implementation reuse.** Filter output from `pkg/obelisk.BuildConnections` after `SpecWithResolvedCredentials`—do not fork URL logic. Wire optional `state` from cached reconcile observation (not fresh helm status on every GET) to avoid poll storms.

**Kong / exposure.** Host discovery on the HTTP edge (e.g. `GET /discover/v1/projects/{id}`) with rate limiting like the mail gateway. Exclude from Funnel. VPS vs local profile still affects URL shape; include the same `notes` and profile hints as `/connections`.

**Naming.** “Project” overlaps with design-doc G9 “New project creates DB” language. UI and docs must state: **A Project groups instances for connection discovery. It does not create databases.** Provisioning stays at the instance and logical-database level in v1.

**Comprehension.** Operator CUJ: enable instances → create Project → add members → copy discovery base URL + rotate token → configure remote app. Developer CUJ: boot → `GET` with Bearer → parse JSON → set driver URLs; treat the response as secret.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * **Projects UI** under Services: create, rename, delete Project; add/remove instance membership (many-to-many).
  * **Desired state** in `obelisk.yaml`: `projects[]` with stable `id`, `name`, `instance_ids[]`, `discovery_enabled`.
  * **GET-only discovery endpoint** per Project returning filtered `Connections`-compatible JSON (full credential URLs).
  * **Per-Project revocable bearer token** stored in cluster secrets; regenerate from UI; audit token rotation and discovery calls.
  * **Separate auth path** for discovery (not GitHub, not admin `LocalToken`).
  * **Kong route** with rate limiting; discovery disabled when operator turns it off for a Project.
  * **CLI/API parity** for Project CRUD on the admin API (GitHub-gated); discovery remains token-only.

* **Out-of-Scope Non-Goals:**
  * Container self-registration or TTL-based membership.
  * Proxying traffic through the discovery service (Pattern B gateway).
  * Consul/Eureka compatibility or generic SD watches.
  * **Option B provisioning** (auto-create Postgres role/DB per Project) in v1—label-only over existing instances.
  * Per-Project scoped credentials that hide sibling databases on a shared instance (shared instance user/password may still see all DBs on that engine until Option B exists).
  * Multiple discovery tokens per Project with fine-grained scopes (one token per Project in v1).
  * Public exposure via Tailscale Funnel.
  * Horizontal scaling / multiple instances of the same service type inside one Project entry (membership is by instance id).

## Alternatives Weighed

| Alternative | Outcome |
| --- | --- |
| **Status quo:** copy per-instance strings from Services; admin-only `/connections` | Rejected by author: too many env vars; remote runtimes cannot use GitHub login. |
| **Rename feature “Connection bundles” in UI** | Rejected by author: keep the label **Projects** in Services workflow; mitigate confusion with glossary and UI helper text. |
| **Endpoints-only discovery** (host/port, no passwords) | Rejected by author: remote apps should receive full URLs from one GET. |
| **Token only in `Authorization` header** | Partially accepted: header is default; URL/path token optional if implementer adds it with documented risk. |
| **Extend dormant `postgres_db` / `redis_db` without `instance_ids`** | Rejected: ambiguous with multiple Postgres instances; replaced by explicit instance membership. |
| **External Consul/etcd on the appliance** | Rejected by author: YAGNI for single-node; declarative Obelisk state is the registry. |
| **Make discovery identical to admin `LocalToken`** | Rejected: blast radius too large; project tokens must be scoped. |
| **M:N deferred to v2 (bundles only 1:1)** | Rejected by author: same instance in multiple Projects required in v1. |

## Key Decisions & Staging for Next Steps

* **Projects = discovery groups** with `instance_ids[]`; retire unused `postgres_db` / `redis_db` on the `Project` type (no production UI depended on them).
* **v1 is label-only:** Projects filter existing instances; no automatic DB/role creation.
* **Discovery response** reuses `BuildConnections` output shape, filtered; include profile `notes` and optional cached `state` per instance.
* **GET only** on the discovery route; all mutation via GitHub-gated admin API.
* **Tokens** in K8s secrets; hashed or raw storage per existing gateway convention; never in public blueprint YAML.
* **Route shape:** stable opaque Project `id` in path (e.g. `/discover/v1/projects/{id}`); Bearer required.
* **Security policy:** document intentional creds-over-HTTP for token holders; rate limits; audit log on discovery success/failure; operator-visible blast-radius warning in UI.
* **G9:** this feature does not close per-app Postgres role isolation (Option B); that remains a separate reconcile effort if pursued later.
* **Goldfish brainstorm** completed 2026-11-19 across design, codebase, zoom-out, comprehension, implementation-readiness, and structural lenses.

**Open for `/egm-draft`:** exact JSON schema version field; whether query-string token is shipped in v1; Kong plugin parity with mail gateway limits; DELETE project behavior when instances still exist in spec.