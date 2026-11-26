"use client";

import { STATE_DOT_COLOR } from "@/components/connectivity/connectivity-format";
import type { ConnectivityPeer } from "@/lib/types";

const MAX_NODES = 8;

const VB_W = 280;
const LABEL_MAX = 13;
const DOT_X = 96;
const HUB_X = 246;
const HUB_R = 18;

function shortName(id: string) {
  return id.length > LABEL_MAX ? `${id.slice(0, LABEL_MAX - 1)}…` : id;
}

/**
 * Flat flow diagram: peers as a quiet column of status dots, hairline curves
 * into one hub. A short bright "comet" glides along each live link toward the
 * hub. Offline links go still and dim.
 */
export function PeerFlowScene({ peers, connected }: { peers: ConnectivityPeer[]; connected: number }) {
  const shown = peers.slice(0, MAX_NODES);
  const extra = peers.length - shown.length;
  const rows = Math.max(shown.length, 3);
  const rowH = rows <= 3 ? 26 : rows <= 5 ? 22 : 18;
  const height = rows * rowH + 16;
  const cy = height / 2;
  const ringColor = peers.length === 0 ? "var(--muted-foreground)" : "var(--success)";
  const yOf = (i: number) => 8 + rowH * (i + 0.5) + ((rows - shown.length) * rowH) / 2;
  const curve = (y: number) =>
    `M ${DOT_X + 8} ${y.toFixed(1)} C ${DOT_X + 56} ${y.toFixed(1)}, ${HUB_X - HUB_R - 52} ${cy.toFixed(1)}, ${HUB_X - HUB_R - 3} ${cy.toFixed(1)}`;

  return (
    <svg viewBox={`0 0 ${VB_W} ${height}`} className="h-auto w-full overflow-visible" aria-hidden="true" focusable="false">

      {/* links */}
      {shown.map((p, i) => {
        const y = yOf(i);
        const d = curve(y);
        const live = p.state === "connected" || p.state === "degraded";
        const color = STATE_DOT_COLOR[p.state];
        return (
          <g key={p.id}>
            <path d={d} fill="none" stroke={color} strokeOpacity={live ? 0.28 : 0.14} strokeWidth={1} strokeLinecap="round" strokeDasharray={live ? undefined : "2 4"} />
            {live && (
              <path
                d={d}
                pathLength={100}
                fill="none"
                stroke={color}
                strokeWidth={1.8}
                strokeLinecap="round"
                strokeDasharray="9 91"
                className="conn-comet"
                data-speed={p.state === "degraded" ? "slow" : undefined}
                style={{ animationDelay: `-${((i * 0.55) % 3).toFixed(2)}s` }}
              />
            )}
          </g>
        );
      })}
      {peers.length === 0 &&
        [0, 1, 2].map((i) => {
          const y = yOf(i);
          return <path key={i} d={curve(y)} fill="none" className="stroke-foreground" strokeOpacity={0.12} strokeDasharray="2 5" />;
        })}

      {/* peers */}
      {shown.map((p, i) => {
        const y = yOf(i);
        const color = STATE_DOT_COLOR[p.state];
        const live = p.state === "connected" || p.state === "degraded";
        return (
          <g key={p.id}>
            <title>{`${p.id} · ${p.state === "never" ? "waiting for first heartbeat" : p.state}`}</title>
            <text x={DOT_X - 10} y={y + 3.5} textAnchor="end" fontSize={10.5} fill="currentColor" className="text-muted-foreground" opacity={live ? 1 : 0.6}>
              {shortName(p.id)}
            </text>
            {live && <circle cx={DOT_X} cy={y} r={7} fill={color} opacity={0.14} />}
            <circle cx={DOT_X} cy={y} r={3.4} fill={live ? color : "var(--card)"} stroke={color} strokeWidth={live ? 0 : 1.3} />
          </g>
        );
      })}
      {peers.length === 0 &&
        [0, 1, 2].map((i) => (
          <circle key={i} cx={DOT_X} cy={yOf(i)} r={3.4} fill="none" className="stroke-foreground" strokeOpacity={0.25} strokeDasharray="1.5 2.5" />
        ))}
      {extra > 0 && (
        <text x={DOT_X - 10} y={height - 1} textAnchor="end" fontSize={9.5} fill="currentColor" className="text-muted-foreground" opacity={0.7}>
          +{extra} more
        </text>
      )}

      {/* hub: a quiet circle whose border flows like the links */}
      <circle cx={HUB_X} cy={cy} r={HUB_R} fill="var(--card)" />
      <circle cx={HUB_X} cy={cy} r={HUB_R} fill="none" stroke={ringColor} strokeOpacity={peers.length ? 0.22 : 0.15} strokeWidth={1} />
      {peers.length > 0 && (
        <circle
          cx={HUB_X}
          cy={cy}
          r={HUB_R}
          fill="none"
          stroke={ringColor}
          strokeOpacity={0.85}
          strokeWidth={1.4}
          strokeLinecap="round"
          strokeDasharray="3 5"
          className="conn-flow"
        />
      )}
      <text x={HUB_X} y={cy + 5} textAnchor="middle" fontSize={14} fontWeight={500} fill={peers.length ? "var(--foreground)" : "var(--muted-foreground)"}>
        {peers.length === 0 ? "0" : connected}
      </text>
    </svg>
  );
}

