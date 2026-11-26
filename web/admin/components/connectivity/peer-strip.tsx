"use client";

import type { ConnectivityState } from "@/lib/types";
import { STATE_LABEL, STRIP_COLOR } from "@/components/connectivity/connectivity-format";

type PeerStripProps = {
  peerId: string;
  bins: ConnectivityState[];
  windowStart?: string;
  binMinutes: number;
};

/** Thin cells, one per time bin, oldest on the left. Each cell has a tooltip. */
export function PeerStrip({ peerId, bins, windowStart, binMinutes }: PeerStripProps) {
  const startMs = windowStart ? new Date(windowStart).getTime() : NaN;
  return (
    <div
      role="img"
      aria-label={`${peerId}: last 24 hours availability`}
      className="flex h-4 w-full gap-px overflow-hidden rounded-sm"
    >
      {bins.map((state, i) => {
        const when = Number.isFinite(startMs)
          ? new Date(startMs + i * binMinutes * 60_000)
          : null;
        const time = when
          ? when.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
          : "";
        return (
          <span
            key={i}
            title={time ? `${time} · ${STATE_LABEL[state]}` : STATE_LABEL[state]}
            className="h-full min-w-px flex-1"
            style={{ backgroundColor: STRIP_COLOR[state] }}
          />
        );
      })}
    </div>
  );
}
