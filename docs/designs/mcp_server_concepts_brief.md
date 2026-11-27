# MCP Gateway

* **Author:** hsoni92
* **Date:** 2026-11-27

## Overview

Obelisk gets a third Exposure gateway next to OpenRouter and Mail: an **MCP gateway**. It is one HTTP endpoint (`{origin}/mcp`) that any of the operator's AI editors and agents can connect to with a single bearer token.

Today, every machine needs its own MCP servers installed and configured. With the gateway, the operator adds the URL and token to Cursor (or Claude Code, or any other MCP client) once, and the agent has the same tools on every machine with nothing installed locally.

v1 ships two built-in tools:

* `web_search`: calls the SearXNG instance already running in Obelisk.
* `send_mail`: sends mail through the mailboxes already configured in the Mail gateway.

Tools are plain Go values in a small registry, so adding a tool later is one function. The admin can enable the gateway, rotate its token, and switch each tool on or off from the Exposure page.

All clients are the operator's own. There are no per-user identities.

## Analysis

Seven independent reviewers read the concept against the codebase. Their findings, with duplicates merged:

**1. `send_mail` is the main risk, and the concept had no answer for it.** An agent reads untrusted web text through `web_search`, then can mail anything it can read, from the operator's real mailbox. Cursor's per-tool approval prompt does not help, because many users turn on auto-run. Kong's rate limit is per IP, so it does not bound a leaked token. The reviewers agree on a layered defense that needs no per-call human approval:

* `send_mail` is off by default. The admin turns it on.
* The reviewers proposed that the server, not the tool description, enforce who can receive mail (an owner address or an admin allowlist). **The author chose to allow any recipient**, so the remaining limits below carry the whole defense.
* One recipient, plain text, small body (about 20-64 KB), no HTML, no attachments.
* An in-app cap per token (for example 5-10 per hour, 20 per day), a rule against line breaks in subject and recipient, and a ten-minute duplicate check.
* A stored, size-limited send log shown in the admin panel: time, recipient, subject, size, body hash.
* Optional: auto-disable `send_mail` after K sends in M minutes until an admin re-enables it.

With any recipient allowed, the worst case is a bounded number of short plain-text mails to any address, each one visible in the log. The caps, not the recipient, set that bound, so they should start low.

**2. Where it runs.** The controller pod runs as root with wide cluster access, and `mail_gateway.md` already rejected a separate pod. Most reviewers favor a new listener (`:8083`) inside the controller, in a new package (`internal/mcpgateway`) that imports no Kubernetes or controller code and reaches its backends only through two narrow interfaces (`Searcher`, `Mailer`). Moving it into its own pod later then costs one `main.go`. Mounting `/mcp` on the mail listener avoids a new port but ties MCP to the mail gateway's on/off state.

**3. Transport.** Kong buffers responses, which breaks streaming (SSE), and the existing mail rate limit (30 per minute per IP) would be used up by protocol traffic (`initialize`, `tools/list`, `ping`). The proposal is a stateless, JSON-only handler: POST returns JSON, notifications return 202, GET and DELETE return 405, no session id. Tool failures return as `isError` results with stable codes; auth and transport failures stay HTTP 401/503. The trade-off: a future streaming tool would force sessions. Nobody has tested whether Cursor accepts this form, so a prototype against Cursor, Claude Code and MCP Inspector comes first.

**4. Token.** Use a separate MCP token with its own credentials file and a distinct prefix (for example `obk_mcp_`), not the mail token. The token sits in plain text in each machine's `mcp.json`, and rotation cuts off every machine at once with no grace period. Store the credentials file as a list from the start so per-client tokens can be added later without a breaking change. Check the token in the app, not with a Kong auth plugin: a Kong plugin would put the token in a ConfigMap and restart Kong on every rotation. Return 401 even when the gateway is disabled, so the endpoint's state is not revealed.

**5. Reachability and TLS.** This is unresolved and blocks the "any machine" goal. Kong's proxy listens on plain HTTP, and the admin's gateway URL is built as `http://<PublicHost>:9080/...`. A bearer token for search and mail would cross the network in clear text unless HTTPS or Tailscale sits in front. On the `local` profile there may be no public path at all (Funnel only allowlists RustFS S3). The panel should not show a copyable config until the design says how HTTPS or the tailnet is guaranteed.

**6. Backends have their own lifecycle.** SearXNG is optional, off by default, and can have several instances. Mail has no pool until a mailbox is added. The admin toggle expresses intent; the panel shows live status per tool (ready, backend missing, backend down). **The author chose to always list both tools**, whether or not they are enabled or their backend is ready. A tool that cannot run returns an `isError` result with a plain message the agent can show to the user (for example "SearXNG is not running; enable it in Services" or "Mail gateway is disabled or has no mailbox"). The reviewers had proposed hiding such tools. The cost of listing: an agent may try a tool that cannot work, and a disabled `send_mail` stays visible to the agent. The message text therefore has to say what to fix and where, and must not reveal secrets or internal addresses. `web_search` picks a running SearXNG instance automatically (the first enabled one) and needs no setting. MCP depends on SearXNG but does not turn it on.

**7. Code changes that come with it.**

* The mail send path is tied to `handleSend`. It must be extracted into an exported `Send(ctx, SendRequest)` so MCP and HTTP share one path. This touches shipped code and needs tests. Calling `:8082` from inside the controller would add a second token, a second hop and the Kong limit.
* The mail code uses only the first recipient and silently drops the rest, so the tool takes a single `to`.
* `buildMIME` (`internal/mail/smtp_send.go`) puts the raw subject into headers. A subject with a line break, written by a model, can inject headers. Reject line breaks in the MCP layer, and preferably fix `buildMIME` for both gateways.
* Kong: add an `/mcp` route in both base files (`deploy/helm/platform/files/kong-dbless.yaml` and `deploy/kong/kong-dbless.yaml`). `kongServiceEqual` compares plugin names only, so later changes to rate or size limits do not reach a running cluster. Decide whether to fix that comparison or accept a Helm upgrade for tuning.
* A new controller port means Helm changes (arg, containerPort, Service port, Kong target). Controller and chart must upgrade together.
* `mail_gateway.md` says only Kong can reach controller `:8082`, but no NetworkPolicy exists in `deploy/`. Add one, or state the gap and enforce limits in the app.
* Blueprint backup (`internal/blueprint/sidecars.go`) needs a field for the MCP token. Without it, a restore leaves the gateway enabled with no token. `Reload()` should leave it disabled when the token is empty. Older bundles without the field must still load.
* The admin UI's `GatewayCard` assumes three steps and an accounts list, and the Exposure page is a fixed two-column grid. Either make the accounts part optional or build a lighter card with a tool-toggle list.
* SearXNG output is large (5-15k tokens of raw JSON). Return the top 5-8 results as title, URL and a snippet of about 300 characters, under a total cap of about 4 KB, with a 10-15 s timeout. Wrap and delimit results as untrusted text. Check `format=json` on a live instance (a 403 means JSON is disabled).
* Observability: today there is only a success log line for mail. Add a structured `mcp_call` line (tool, outcome, duration, request id; never bodies or token), a rate-limited auth-failure log, a runtime status block, and a `syshealth` check.

**8. Registry size.** One reviewer wants a fixed list of tools. Others want a small tool contract (name, description, input schema, call, risk level). Agreed middle: tools are plain Go values registered explicitly, with no plugin loading or reflection. The spec stores only on/off per tool name. Unknown names in the spec are kept on save. Side-effecting tools default to off. The admin UI reads the tool list from the API, so a new tool needs no front-end change.

**Effort.** Roughly 300-500 lines of Go plus a copied panel. A hand-written stateless handler is about 150-250 lines using `encoding/json`. The official Go SDK in JSON mode would add a dependency; decide after the Cursor prototype.

**Ideal Final Result.** The operator opens Exposure, enables the MCP gateway, sees an HTTPS endpoint and a token shown once, and copies a ready-made `mcp.json` snippet. Pasting it into Cursor once gives search, and mail when allowed, on any machine. Each tool row shows live backend status. A leaked token or an injected agent can cause only a small, visible amount of harm, and the send log shows what happened.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * New MCP gateway in the Exposure section, served by `obelisk-controller` on its own listener, routed through Kong at `/mcp`.
  * One bearer token, separate from the mail and OpenRouter tokens, checked in the app, with immediate rotation.
  * Stateless, JSON-only MCP: `initialize`, `ping`, `tools/list`, `tools/call`.
  * Two built-in tools: `web_search` (SearXNG) and `send_mail` (existing mail send path).
  * Tool registry in Go; per-tool on/off in the spec and admin UI; side-effecting tools off by default.
  * `send_mail` limits: any recipient allowed, one recipient per call, plain text, size cap, per-token cap, duplicate check, stored send log.
  * Live per-tool backend status, a Test button, and a copy-ready `mcp.json` snippet in the panel.
  * Extract `Send` from the mail gateway handler as a prerequisite.
  * Token included in blueprint backup; spec, schema, Helm, Kong, dev scripts, `syshealth`, and a runbook.
* **Out-of-Scope Non-Goals:**
  * Proxying remote MCP servers (GitHub, GitLab, Bitbucket, Buildkite). Possible later.
  * OAuth and OAuth discovery for MCP clients.
  * Streaming (SSE), sessions, MCP resources and prompts.
  * Per-client tokens and scopes (the file format leaves room; the feature is not built).
  * Federation through the peer gateway.
  * Fetching web pages; `web_search` returns snippets only.
  * Content scanning of mail bodies.
  * Auto-enabling SearXNG or the mail gateway.

## Alternatives Weighed

* **Install MCP servers locally on each machine (status quo).** Rejected by author: this is the pain being solved.
* **Proxy remote-service MCP servers (GitHub, GitLab, Bitbucket, Buildkite) in v1.** Rejected by author: it needs OAuth, third-party token storage, tool-count limits in editors, and a larger blast radius. Deferred.
* **Separate pod for the MCP server.** Not chosen for v1: `mail_gateway.md` rejected a separate pod, and a pod needs its own credentials mount. The package boundary keeps this option cheap later.
* **Mount `/mcp` on the existing mail listener.** Not chosen by most reviewers: it ties MCP lifecycle to the mail gateway's enabled flag. Rejected by author's decision (mail flag gates only `send_mail`, so search must not depend on the mail listener).
* **Reuse the mail gateway token.** Rejected by reviewers: one leaked mail token would then also give search and mail through MCP, and rotating one would break the other.
* **Kong `key-auth` plugin for the token.** Rejected by reviewers: the token lands in a ConfigMap and rotation restarts Kong.
* **Call the mail gateway over `:8082` from the MCP code.** Rejected by reviewers: adds a second token, a second hop, and the Kong rate limit.
* **Streaming transport with sessions.** Rejected for v1: Kong buffering, extra state, no current tool needs it.
* **Official MCP Go SDK.** Undecided: avoids hand-written protocol code but adds a dependency; decide after the prototype.
* **Per-call human approval for `send_mail`.** Rejected by reviewers: clients often auto-run tools, so it is not a dependable control.

## Key Decisions & Staging for Next Steps

**Decided with the author:**

* v1 has two internal tools only: mail and SearXNG search. Remote service proxying is dropped.
* Single bearer token; credentials stay on the Obelisk server.
* All clients are the operator's own; "some control" means admin on/off switches per tool.
* Lightweight placement inside the existing controller is acceptable.

**Decided by the author in the second round:**

1. **Recipients: anyone.** There is no recipient allowlist and no owner-only rule. The server still enforces one recipient, plain text, a size cap, a per-token hourly and daily cap, a line-break check, a duplicate check, and a visible send log. With recipients open, these limits and the log are the only brake on an injected agent. The brief therefore keeps them as required, not optional. `send_mail` stays off by default, and the admin turns it on.
2. **Reachability follows the existing pattern.** On a `local` machine, the endpoint is reached over Tailscale on the same tailnet. On a `vps`, it uses the public URL the other Exposure services use. **Residual risk:** the other services' URL is built as `http://<PublicHost>:9080/...`. If the VPS public URL is plain HTTP, the bearer token crosses the internet in clear text. Tailscale traffic is encrypted. The design must check how the VPS URL is served today. If it is HTTP, the design either accepts that for v1 and says so in the panel and runbook, or fronts `/mcp` with TLS.
3. **`send_mail` requires the mail gateway's enabled flag**, as well as at least one ready mailbox. If either is missing, the tool is still listed but returns a message saying what is missing. `web_search` does not depend on the mail gateway.
5. **Tools are always listed; backends are found automatically.** `web_search` uses a running SearXNG with no configuration, and shows a clear message when none is running. `send_mail` does the same for mail. Disabled-by-admin tools also stay listed and return a "disabled by admin" message.
4. **Placement: a new listener (`:8083`) inside the controller**, in a package with no Kubernetes imports. Reasoning: a path on the mail listener would make search depend on the mail gateway's state, which conflicts with decision 3. A separate pod adds a credentials mount and a second deployment for two small tools. Moving to a pod later costs one `main.go`.

**Still open (the design draft can propose defaults):**

6. Sender mailbox: round-robin pool (default proposal, same as the mail gateway), or one account pinned for MCP.
7. Handwritten handler or official SDK, after a prototype against Cursor, Claude Code and MCP Inspector.
8. Token storage: hashed at rest, or plain text like the mail token. Include in project discovery (`exposure/mcp`)? (Reviewers lean no.)
9. Send log: stored on disk with bounded size (default proposal), or in memory only; whether subjects and bodies are kept.
10. Fix Kong plugin-config drift in `kongServiceEqual` now, and add the missing NetworkPolicy now, or document both gaps.
11. UI: extend `GatewayCard` or build a lighter tool-toggle card; CLI parity with the Exposure page.

**Suggested slices:**

1. Extract `mailgateway.Send` (with tests) and fix `buildMIME` header handling.
2. `internal/mcpgateway`: registry, stateless handler, token check, the two tools; prototype against Cursor.
3. Spec, schema, credentials file, blueprint field, Helm port, Kong route in both base files, dev scripts.
4. Admin API (`GET/PATCH /api/v1/mcp/config`, `POST /api/v1/mcp/test`) and UI card.
5. Observability (`mcp_call` log, `syshealth`), runbook `docs/runbooks/mcp-server.md`, SITEMAP entry.

**Coverage limits of this review:** no reviewer read all of `obelisk.md`, `peer_gateway.md`, `internal/openroutergateway/*`, or every web panel component. Nobody ran code or tested Cursor against a stateless endpoint.
