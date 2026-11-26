# Connectivity page and gateway peer status

* **Author:** hsoni92
* **Date:** 2026-11-26
* **Status:** Implemented. Brief revised after author review (round 2).

## Overview

Obelisk's Tailscale controls sit inside Platform settings, under a tab called "Tailscale & gateway". Operators have to dig to find them, and the page does not show whether the gateway can actually reach its peers.

This brief proposes a top-level **Connectivity** page in the admin UI. It has three parts:

1. **Tailscale connect and status** for every host, moved out of Platform settings.
2. **Gateway peer view** on a VPS that runs as a gateway. For each local Obelisk peer it shows two signals: whether the peer has recently sent a heartbeat, and whether the gateway can reach it over the tailnet. It also shows the current state and how long it has held.
3. **24-hour availability strip** per peer, built from state changes stored on the VPS.

The operator question this answers: "Is my home box reaching the VPS, can the VPS reach it back, and has it dropped in the last day?"

This brief owns the Connectivity page for now. `docs/designs/cloudflare_tunnel.md` also plans a Connectivity page; that conflict will be resolved when that design is revisited, not here.

## Analysis

### Two signals per peer

| Signal | Direction | What it proves | Source |
| --- | --- | --- | --- |
| **Heartbeat** | Peer to gateway | The peer is running and can reach the gateway. | The peer's existing advertise request. The gateway sets `LastSeen`. |
| **Reachability probe** | Gateway to peer | The gateway can open a connection to the peer over the tailnet. | New. A background TCP connect from the gateway. |

Each signal can be true without the other. A peer can heartbeat fine while the gateway cannot reach it (for example, a host firewall blocks the gateway), and the reverse is possible after a gateway restart. The page shows both, so an operator can see which side broke.

### Peer states

The page shows one state per peer, derived from the two signals:

| State | Heartbeat | Probe | Meaning |
| --- | --- | --- | --- |
| **Connected** | fresh | ok | Both directions work. |
| **Degraded** | fresh | failing | Peer is talking to the gateway, but the gateway cannot reach it. Client traffic through the gateway will likely fail. |
| **Offline** | stale | not shown | No heartbeat within the threshold. Probe result is shown only as its last known value. |
| **Never connected** | none | none | Admitted peer that has never advertised. |
| **No data** | — | — | The gateway itself was not running, so nothing was observed. Never shown as a peer outage. |

### Probe design

* **Target:** the peer's tailnet address and the admin HTTP port it advertises. The advertise payload must be checked to confirm both fields are present. If the port is not advertised today, the peer's advertise request needs a small addition, and that is a prerequisite.
* **Method:** a TCP connect only. No request body, no auth, no data sent. It proves the path and the port are open, not that the service behind it is healthy.
* **Timing:** one probe per peer every 60 seconds, each with a 5-second timeout. The 5 seconds matches the nginx connect timeout in `peer_gateway.md` (OQ5).
* **Load:** probes run in one background loop, with at most 4 running at once and a small random offset per peer, so a gateway with many peers does not open many connections at the same moment.
* **Hysteresis:** a peer moves to Degraded after 2 failed probes in a row, and back to Connected after 2 successful probes in a row. This stops a single dropped packet from flipping the state.
* **Isolation:** probe results are display data only. The probe never calls route programming or reconcile. Routing behaviour stays as it is today.
* **Gating:** the loop does nothing unless the host is a gateway, checked on each tick, because the role can change at runtime.

### Background loop and history

One background loop on the gateway does three things every 30 seconds:

1. Marks peers whose heartbeat is stale.
2. Runs the probes that are due.
3. Writes a state transition to the history store when a peer's combined state changes. Transitions are not written on every tick.

The history store is the existing `internal/history` store, with a new `peer` kind. It already keeps 24 hours of records and loads them on startup. Reusing it keeps one store for state history. The earlier overview design chose memory rings to avoid disk writes, but this is a small number of transition events, not per-second samples, so a disk-backed store fits.

Known cost: the store rewrites its whole file on each write. That is fine for a handful of peers with few transitions. The brief sets a cap of 50 peers per gateway. If transitions become frequent, a later change can move this to an append-only file.

### Gateway liveness marker

To tell "peer was down" from "gateway was down", the gateway writes a small "last alive" timestamp to its own file every 60 seconds. This file is overwritten atomically, not appended. When the page renders history, any gap longer than 2 minutes in the marker is shown as **No data**. Uptime is counted only over time the gateway was alive.

### Uptime

Each peer record stores `state_since`, the time its current combined state began. It is updated on each transition.

* Uptime shown is the time since `state_since`, counted only while the gateway was alive.
* For peers that exist before this feature ships, there is no history. The page shows "observed since <release date>", never a full streak.

### 24-hour strip

* 24 hours in 15-minute bins, so 96 cells per peer. One minute resolution would be about 1,440 cells, which the UI cannot show legibly.
* Each bin takes the worst state it contained. Order from worst to best: Offline, Degraded, Connected. A bin with no gateway liveness is **No data**.
* The strip is labelled with its window ("last 24 h") and has a legend. "No data" uses a colour distinct from "Offline".

### Prerequisites from the review

These must land before or with this feature:

* **One stale threshold constant** (2 minutes for heartbeat) used everywhere. The advertise interval is documented in three different ways today (30s, 45s, "about 30s"). The brief sets 30 seconds and fixes the comments and copy to match.
* **Registry safety:** one process-wide owner with a single lock, atomic saves using a unique temporary file, and no writes from read-only requests. A failed load must keep the last good state and must never program an empty peer set.
* **Verify before relying on it:** the review reported that a peer flap may restart the gateway's Kong and nginx. This is single-source and unverified. Whether or not it holds, the new loop never calls route programming, so the new feature does not depend on the answer.

### Scope and layout by host type

* **Gateway VPS:** Tailscale status, gateway role control, peer list with both signals, state, uptime, and the 24-hour strip.
* **Local host (non-gateway):** Tailscale status, the role control (so the host can be promoted to gateway), and the local registration card (advertise URL, push now, last status, unregister). No peer list.
* **Standalone VPS:** Tailscale is not used on this profile. The page shows an explanation and the gateway role control only.

The backend returns one capability field, for example `peer_view_enabled`, so the client does not derive the layout from two separate fields.

### Vocabulary

The UI and docs use one set of words:

* **Connected, Degraded, Offline, Never connected, No data** for the peer states above.
* **Heartbeat** and **Reachable** for the two signals.
* **Uptime** only on the Connectivity page, and labelled "in current state".
* **Local** for a peer Obelisk, and **Gateway** for the VPS. Avoid "home Obelisk" and "appliance" on this page.

### Navigation and migration

* New top-level nav item **Connectivity** at `/connectivity`.
* The Tailscale and gateway controls move off Platform. Platform keeps Appearance and Blueprint, and its subtitle no longer mentions Tailscale.
* `/platform#tailscale` redirects to `/connectivity` for one release, so old bookmarks work.
* `peer_gateway.md` says the role is chosen in Platform settings, and plans a separate Peers navigation entry. This brief moves both to Connectivity. That design should be updated when this ships.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  * Top-level Connectivity page with the Tailscale section, visible on every host.
  * Gateway peer view with heartbeat and reachability per peer, the combined state, and time in state.
  * 24-hour per-peer strip in 15-minute bins, with "No data" for gateway downtime.
  * A lightweight reachability probe: TCP connect, 60-second interval, bounded concurrency, hysteresis.
  * State transitions stored in the existing history store (kind `peer`).
  * Redirect from the old Platform hash.
* **Out-of-Scope Non-Goals:**
  * Probing the actual services (Postgres, Redis, Kong routes). The probe proves the path is open, nothing more.
  * Any change to route programming or failover. The probe is display only.
  * Backfilling history before release.
  * Multi-gateway or failover behaviour (already excluded by `peer_gateway.md`).
  * Fixing the auto-admit token re-issue (see Open Risks). It is a security issue and gets its own change.
  * The animated dotted-line SVG in the first release. It follows once the data is stable.
  * Changing `cloudflare_tunnel.md`.

## Alternatives Weighed

* **Heartbeat only, no probe.** Rejected by the author in favour of the probe. Kept in the design as the fallback if the probe proves unreliable on some networks.
* **Separate rotated log files for history.** Rejected. It duplicates `internal/history`, and the state changes are sparse, so the existing store is enough.
* **Probe every peer every 10 seconds.** Rejected. Lots of connections for little gain; 60 seconds with hysteresis gives the same accuracy for a display.
* **Probe via a full HTTP health request.** Deferred. A TCP connect is enough for a path check and does not depend on the peer's service code. An HTTP check can be added later as a separate signal.
* **Compute the strip in the browser from live polling.** Rejected. Samples would exist only while an admin tab is open.
* **Full animated SVG in v1.** Deferred. The motion is decorative. The list carries the information.

## Key Decisions & Staging for Next Steps

Decided by the author:

* Connection state is based on heartbeat, and a reachability probe is added as a second signal.
* History is kept on the VPS, in lightweight local data. Sampling starts at release. No backfill.
* Connectivity is a top-level navigation item.
* `cloudflare_tunnel.md` is out of scope for this change.

Decided in this revision (best judgement, open to change):

* Probe design as above (TCP, 60s, 5s timeout, max 4 concurrent, 2-probe hysteresis, display only).
* Single 30-second background loop; history reuses `internal/history` with kind `peer`.
* Gateway liveness marker every 60 seconds; gaps over 2 minutes show as "No data".
* Strip in 15-minute bins, worst state wins.
* Pre-release uptime shows "observed since <release date>".
* Peer history is not included in backups or blueprint exports. It is short-lived operational data, rebuilt from live state.
* Cap of 50 peers per gateway.
* Duplicate or restored machine IDs are detected and shown as a warning on the peer row. They are not silently merged.
* Rollout ships without a feature flag. The new response fields are additive, so rollback is a UI-only change.

Staging:

1. **Prerequisites:** one stale constant and corrected advertise-interval copy; registry lock and atomic saves; error handling on load.
2. **Advertise check:** confirm the peer's advertise payload includes its tailnet address and admin port. Add them if missing.
3. **Background loop:** heartbeat staleness, probes, transition writes, and the gateway liveness marker. Gated on the gateway role each tick. Unit tests for hysteresis and for gap handling.
4. **Connectivity page:** Tailscale section, role control, local registration card, and the static peer list with both signals and state. Redirect the old Platform hash.
5. **Uptime and the 24-hour strip** from stored transitions, with "No data" gaps and the window labelled.
6. **Animated SVG:** decorative only, honours reduced-motion settings, with all information also in text.

## Open Risks

* **Probe reliability on some networks** (medium): tailnet paths can be slow to connect. The 5-second timeout and hysteresis reduce false alarms. Watch for false Degraded states after rollout.
* **Probe target may be missing from the advertise payload** (medium, to verify): stage 2 checks this first.
* **Auto-admit token re-issue** (high, security, out of scope here): with auto-admit on, a tailnet member who knows a peer ID can get that peer's token re-issued. This should be bound to the recorded identity or at least audited. It needs its own change; this page makes it more visible but does not cause it.
* **Route restart on peer flap** (high, unverified): if confirmed, the existing reconcile path is still a problem for any future display-driven change. This feature avoids it by design.
* **Registry races** (high): a stale save can roll back a healthy peer's heartbeat. Fixed as a prerequisite.
* **History store rewrite cost** (low at current scale): the cap of 50 peers keeps it bounded. Revisit if transitions get frequent.
* **Vocabulary drift** (low): the old docs use "reachable", "unhealthy" and "stale". The new terms should be used in the UI and docs together.

## Implementation status

Built in the working tree on branch `remove-leaked-test-key`. Not committed.

| Stage | What was built | Where |
| --- | --- | --- |
| 1. Prerequisites | One process-wide registry per file with a single lock; atomic save (unique temp file, fsync, rename); reachability computed at read time; no writes from GET requests; one `HeartbeatStaleAfter` (2 min) and `AdvertiseInterval` (30 s) constant; a failed registry load skips exposure reconcile instead of programming an empty set | `internal/peer/registry.go`, `internal/controller/scheduler.go`, `internal/controller/peer_exposure.go` |
| 2. Advertise check | Confirmed: advertise payload already carries `tailnet_host` and `http_backhaul_port`. The probe falls back to the admin HTTP port when the backhaul port is absent. | `internal/peer/types.go`, `internal/peer/monitor.go` |
| 3. Background loop | Heartbeat check, TCP probe per peer (60 s, 5 s timeout, max 4 concurrent, 0-10 s jitter), hysteresis (2 failures to degrade, 2 successes to recover), transition events in `internal/history` (kind `peer`), gateway liveness marker, gap and checkpoint events. Gated on gateway role each tick. Never calls route reconcile. | `internal/peer/monitor.go`, `internal/peer/connectivity.go`, `internal/history/store.go`, `cmd/obelisk-controller/main.go` |
| 4. Connectivity page | New top-level nav item and route. Tailscale card and gateway role control moved off Platform. Peer list now lives on Connectivity. `#tailscale` on Platform redirects. | `web/admin/app/(console)/connectivity/`, `web/admin/components/connectivity/`, `web/admin/components/platform/` |
| 5. Uptime and strip | Uptime is observed time in the current state, excluding gateway gaps. 24 h in 15-minute bins, worst state wins, "No data" where the gateway was not observing. Pre-release or pruned history is shown as a lower bound (`≥`). | `internal/peer/connectivity.go`, `internal/adminapi/connectivity.go` |
| 6. Animated SVG | Decorative dotted flow from each peer into the gateway hub, animated only for live peers. Respects `prefers-reduced-motion`. All information is also in the text list. | `web/admin/components/connectivity/peer-flow-diagram.tsx`, `web/admin/app/globals.css` |

Deviations from the brief:

* The liveness marker is written on every 30-second monitor tick, not every 60 seconds. Gap detection uses the same marker.
* The Tailscale card comes first on the page, then the peer view.
* The peer cap (50) limits probing and history writes. Peers beyond the cap are still listed, with state from the heartbeat alone.
* The page is not visually checked in a browser. It builds and typechecks, and the handler has tests.

Not done, on purpose:

* Auto-admit token re-issue (see Open Risks). It is a security issue and needs its own change.
* `cloudflare_tunnel.md` and `peer_gateway.md` still describe the old Platform placement. Update them when this ships.

Tests added: `internal/peer/connectivity_test.go` (state rules, gap handling, strip bins, hysteresis, gateway gap on restart) and `internal/adminapi/connectivity_test.go` (local host gets no peer view; gateway peer state and strip). The full Go suite and the race detector pass on the changed packages. `npm run build` succeeds. `tsc` passes. ESLint reports one error in `federation-peers-panel.tsx` (`set-state-in-effect`), which predates this change.
