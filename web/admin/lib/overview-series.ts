import type { OverviewComponent, OverviewHistoryEvent } from "@/lib/types";

export type TimelinePoint = {
  t: number;
  healthy: number;
  attention: number;
  stopped: number;
};

/**
 * Replays state-change events into a bucketed series: how many components were
 * healthy / needing attention / stopped at each point in the window.
 * Components with no event before a bucket are assumed to be in their earliest
 * known state (current state when they have no history at all).
 */
export function buildTimeline(
  components: OverviewComponent[],
  history: OverviewHistoryEvent[],
  hours: number,
  now = Date.now(),
): TimelinePoint[] {
  const buckets = hours <= 1 ? 30 : 48;
  const span = hours * 3_600_000;
  const step = span / buckets;
  const byKey = new Map<string, { at: number; state: string }[]>();
  for (const e of history) {
    const list = byKey.get(e.key) ?? [];
    list.push({ at: new Date(e.at).getTime(), state: e.state });
    byKey.set(e.key, list);
  }
  for (const list of byKey.values()) list.sort((a, b) => a.at - b.at);

  const out: TimelinePoint[] = [];
  for (let i = 0; i <= buckets; i++) {
    const t = now - span + i * step;
    const p: TimelinePoint = { t, healthy: 0, attention: 0, stopped: 0 };
    for (const c of components) {
      const evs = byKey.get(c.key);
      let state = c.state;
      if (evs?.length) {
        state = evs[0].state;
        for (const ev of evs) {
          if (ev.at <= t) state = ev.state;
          else break;
        }
      }
      if (state === "running") p.healthy++;
      else if (
        state === "degraded" ||
        state === "pending" ||
        state === "starting" ||
        state === "stopping"
      ) {
        p.attention++;
      }
      else p.stopped++;
    }
    out.push(p);
  }
  return out;
}

export function relativeTime(iso: string, now = Date.now()): string {
  const s = Math.max(0, Math.round((now - new Date(iso).getTime()) / 1000));
  if (s < 45) return "just now";
  if (s < 3600) return `${Math.round(s / 60)}m ago`;
  if (s < 86400) return `${Math.round(s / 3600)}h ago`;
  return `${Math.round(s / 86400)}d ago`;
}

const MIN_UPTIME_MS = Date.UTC(2000, 0, 1);

/** Elapsed time since an ISO timestamp (uptime-style, no "ago" suffix). */
export function formatDuration(iso: string, now = Date.now()): string {
  const started = new Date(iso).getTime();
  if (!Number.isFinite(started) || started < MIN_UPTIME_MS) return "—";
  const s = Math.max(0, Math.floor((now - started) / 1000));
  if (!Number.isFinite(s) || s <= 0) return "—";
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m`;
  if (s < 86400) return `${Math.floor(s / 3600)}h`;
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  if (d < 14 && h > 0) return `${d}d ${h}h`;
  return `${d}d`;
}

const SEVERITY: Record<string, number> = {
  degraded: 0,
  pending: 1,
  stopped: 2,
  running: 3,
};
export const severity = (state: string) => SEVERITY[state] ?? 2;
