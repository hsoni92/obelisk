import type { ConnectivityPeer, ConnectivityState } from "@/lib/types";

const RANK: Partial<Record<ConnectivityState, number>> = {
  offline: 0,
  degraded: 1,
  connected: 2,
};

/** One cell per time bin: the worst known peer state in that bin. */
export function aggregateStrip(peers: ConnectivityPeer[]): ConnectivityState[] {
  const bins = peers[0]?.strip.length ?? 0;
  return Array.from({ length: bins }, (_, i) => {
    let worst: ConnectivityState | null = null;
    for (const p of peers) {
      const s = p.strip[i];
      const r = RANK[s];
      if (r === undefined) continue;
      if (worst === null || r < (RANK[worst] ?? 99)) worst = s;
    }
    return worst ?? "no_data";
  });
}

/** Share of observed bins in which a peer was connected, across all peers. */
export function availability(peers: ConnectivityPeer[]): number | null {
  let known = 0;
  let up = 0;
  for (const p of peers) {
    for (const s of p.strip) {
      if (RANK[s] === undefined) continue;
      known++;
      if (s === "connected") up++;
    }
  }
  return known ? (up / known) * 100 : null;
}

