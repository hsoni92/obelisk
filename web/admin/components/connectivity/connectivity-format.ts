import type { ConnectivityState } from "@/lib/types";

/** Compact span for uptime and downtime: 45s, 12m, 3h 5m, 2d 4h. */
export function formatSpan(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) {
    const remM = m % 60;
    return remM ? `${h}h ${remM}m` : `${h}h`;
  }
  const d = Math.floor(h / 24);
  const remH = h % 24;
  return remH ? `${d}d ${remH}h` : `${d}d`;
}

export const STATE_LABEL: Record<ConnectivityState, string> = {
  connected: "Connected",
  degraded: "Degraded",
  offline: "Offline",
  never: "Never connected",
  no_data: "No data",
  unknown: "Unknown",
};

/** Colors use the existing theme tokens, so light and dark mode both work. */
export const STRIP_COLOR: Record<ConnectivityState, string> = {
  connected: "var(--success)",
  degraded: "var(--warning)",
  offline: "var(--destructive)",
  never: "var(--muted)",
  no_data: "var(--muted)",
  unknown: "var(--muted)",
};

export const STATE_DOT_COLOR: Record<ConnectivityState, string> = {
  connected: "var(--success)",
  degraded: "var(--warning)",
  offline: "var(--destructive)",
  never: "var(--muted-foreground)",
  no_data: "var(--muted-foreground)",
  unknown: "var(--muted-foreground)",
};
