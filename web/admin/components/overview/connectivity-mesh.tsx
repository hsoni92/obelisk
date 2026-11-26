"use client";

import { STATE_DOT_COLOR, STRIP_COLOR } from "@/components/connectivity/connectivity-format";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeading } from "@/components/ui/card";
import { relativeTime } from "@/lib/overview-series";
import { connectivityPollOptions, useGetConnectivityQuery } from "@/lib/store/obelisk-api";
import type { ConnectivityPeer, ConnectivityState } from "@/lib/types";
import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import { ArrowUpRight, Waypoints } from "lucide-react";
import Link from "next/link";
import { useMemo } from "react";

const W = 360;
const H = 210;
const CX = W / 2;
const CY = H / 2;
const RX = 138;
const RY = 72;
const HUB_R = 24;
const MAX_NODES = 8;
const LABEL_MAX = 12;

const RANK: Partial<Record<ConnectivityState, number>> = {
  offline: 0,
  degraded: 1,
  connected: 2,
};

/** One cell per time bin: the worst known peer state in that bin. */
function aggregateStrip(peers: ConnectivityPeer[]): ConnectivityState[] {
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
function availability(peers: ConnectivityPeer[]): number | null {
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

function nodePosition(i: number, n: number) {
  const start = n === 1 ? -35 : -90 + (n % 2 === 0 ? 180 / n : 0);
  const a = ((start + (360 / n) * i) * Math.PI) / 180;
  return { x: CX + RX * Math.cos(a), y: CY + RY * Math.sin(a), cos: Math.cos(a), sin: Math.sin(a) };
}

/** Curve from a peer into the edge of the hub, bowed sideways so lines do not overlap. */
function linkPath(x: number, y: number) {
  const dx = CX - x;
  const dy = CY - y;
  const len = Math.hypot(dx, dy) || 1;
  const ex = CX - (dx / len) * (HUB_R + 3);
  const ey = CY - (dy / len) * (HUB_R + 3);
  const mx = (x + ex) / 2 - (dy / len) * 16;
  const my = (y + ey) / 2 + (dx / len) * 16;
  return `M ${x.toFixed(1)} ${y.toFixed(1)} Q ${mx.toFixed(1)} ${my.toFixed(1)} ${ex.toFixed(1)} ${ey.toFixed(1)}`;
}

function shortName(id: string) {
  return id.length > LABEL_MAX ? `${id.slice(0, LABEL_MAX - 1)}…` : id;
}

function Constellation({ peers, connected }: { peers: ConnectivityPeer[]; connected: number }) {
  const reduce = useSafeReducedMotion();
  const shown = peers.slice(0, MAX_NODES);
  const hubColor = peers.length === 0 ? "var(--muted-foreground)" : "var(--broadcast)";

  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="h-auto w-full overflow-visible" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id="mesh-hub-glow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor="var(--broadcast)" stopOpacity="0.55" />
          <stop offset="100%" stopColor="var(--broadcast)" stopOpacity="0" />
        </radialGradient>
      </defs>

      {/* orbit guides */}
      <ellipse cx={CX} cy={CY} rx={RX} ry={RY} fill="none" stroke="currentColor" strokeOpacity={0.12} strokeDasharray="2 6" className="text-foreground" />
      <ellipse cx={CX} cy={CY} rx={RX * 0.55} ry={RY * 0.55} fill="none" stroke="currentColor" strokeOpacity={0.07} className="text-foreground" />

      {/* hub glow and ripples */}
      <circle cx={CX} cy={CY} r={HUB_R * 2.6} fill="url(#mesh-hub-glow)" />
      {peers.length > 0 &&
        [0, 1, 2].map((d) => (
          <circle key={d} className="conn-ripple" data-delay={d} cx={CX} cy={CY} r={HUB_R * 1.5} fill="none" stroke={hubColor} strokeWidth={1.25} />
        ))}

      {/* links, pulses, nodes */}
      {shown.map((p, i) => {
        const { x, y, sin } = nodePosition(i, shown.length);
        const live = p.state === "connected" || p.state === "degraded";
        const color = STATE_DOT_COLOR[p.state];
        const d = linkPath(x, y);
        return (
          <g key={p.id}>
            <path d={d} fill="none" stroke={color} strokeWidth={1.4} strokeLinecap="round" strokeDasharray="1.5 5" opacity={live ? 0.85 : 0.3} />
            {live && !reduce &&
              [0, 1].map((k) => (
                <circle key={k} r={k === 0 ? 2.6 : 1.8} fill={color} opacity={k === 0 ? 1 : 0.55}>
                  <animateMotion
                    dur={p.state === "degraded" ? "5s" : "2.6s"}
                    begin={`-${(i * 0.55 + k * 1.3).toFixed(2)}s`}
                    repeatCount="indefinite"
                    path={d}
                  />
                </circle>
              ))}
            <circle cx={x} cy={y} r={11} fill={color} opacity={0.16} />
            <circle cx={x} cy={y} r={5} fill={color} stroke="var(--card)" strokeWidth={1.5} />
            <text x={x} y={y + (sin >= 0 ? 22 : -14)} textAnchor="middle" fontSize={10} fill="currentColor" className="text-muted-foreground">
              {shortName(p.id)}
            </text>
          </g>
        );
      })}

      {/* hub */}
      <g className="conn-spin">
        <circle cx={CX} cy={CY} r={HUB_R + 7} fill="none" stroke={hubColor} strokeOpacity={0.45} strokeWidth={1} strokeDasharray="3 7" />
      </g>
      <circle cx={CX} cy={CY} r={HUB_R} fill={hubColor} />
      <circle cx={CX} cy={CY} r={HUB_R} fill="none" stroke="var(--card)" strokeOpacity={0.6} strokeWidth={1.5} />
      <text x={CX} y={CY + 5} textAnchor="middle" fontSize={16} fontWeight={600} fill="var(--broadcast-foreground)">
        {peers.length === 0 ? "–" : connected}
      </text>
      <text x={CX} y={CY + HUB_R + 22} textAnchor="middle" fontSize={10} fill="currentColor" className="text-muted-foreground">
        gateway
      </text>
    </svg>
  );
}

function HeatStrip({ cells }: { cells: ConnectivityState[] }) {
  return (
    <div role="img" aria-label="Combined availability, last 24 hours" className="flex h-6 w-full gap-px overflow-hidden rounded-md">
      {cells.map((s, i) => (
        <span key={i} className="h-full min-w-px flex-1" style={{ backgroundColor: STRIP_COLOR[s], opacity: s === "no_data" ? 0.7 : 1 }} />
      ))}
    </div>
  );
}

/**
 * Overview card for the gateway peer mesh. Renders nothing until the role is
 * known, and nothing on hosts that are not a gateway.
 */
export function ConnectivityMesh() {
  const { data } = useGetConnectivityQuery(undefined, connectivityPollOptions);
  const peers = useMemo(() => data?.peers ?? [], [data?.peers]);
  const strip = useMemo(() => aggregateStrip(peers), [peers]);
  const avail = useMemo(() => availability(peers), [peers]);

  if (!data?.peer_view_enabled) return null;

  const connected = data.counts.connected ?? 0;
  const degraded = data.counts.degraded ?? 0;
  const offline = data.counts.offline ?? 0;
  const tone = peers.length === 0 ? "outline" : offline || degraded ? "warning" : "success";
  const summary =
    peers.length === 0
      ? "No peers yet"
      : `${connected} of ${peers.length} connected`;

  return (
    <Card className="overflow-hidden">
      <CardHeading
        icon={Waypoints}
        tone="blue"
        title="Connectivity"
        description="Home Obelisks reaching this gateway"
        badge={<Badge variant={tone}>{summary}</Badge>}
        actions={
          <Link
            href="/connectivity"
            className="inline-flex items-center gap-1 text-metadata font-medium text-link hover:underline"
          >
            Open
            <ArrowUpRight className="size-3.5" aria-hidden />
          </Link>
        }
      />
      <div className="grid items-center gap-6 p-4 sm:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)] sm:p-6">
        <div role="img" aria-label={`${summary}. ${degraded} degraded, ${offline} offline.`}>
          <Constellation peers={peers} connected={connected} />
        </div>

        <div className="space-y-5">
          <div className="flex items-end gap-6">
            <div>
              <p className="text-5xl font-light leading-none tabular-nums text-foreground">
                {connected}
                <span className="text-2xl text-muted-foreground">/{peers.length}</span>
              </p>
              <p className="mt-1 text-metadata text-muted-foreground">peers connected</p>
            </div>
            <div>
              <p className="text-3xl font-light leading-none tabular-nums text-foreground">
                {avail === null ? "—" : `${avail >= 99.95 ? 100 : avail.toFixed(1)}%`}
              </p>
              <p className="mt-1 text-metadata text-muted-foreground">available, {data.window_hours} h</p>
            </div>
          </div>

          {peers.length > 0 ? (
            <div className="space-y-2">
              <HeatStrip cells={strip} />
              <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                <span>{data.window_hours} h ago</span>
                <span className="flex items-center gap-3">
                  {degraded > 0 ? <span style={{ color: "var(--warning)" }}>{degraded} degraded</span> : null}
                  {offline > 0 ? <span style={{ color: "var(--destructive)" }}>{offline} offline</span> : null}
                </span>
                <span>now</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Waiting for a home Obelisk to register with this gateway.
            </p>
          )}

          {data.monitoring_since ? (
            <p className="text-[11px] text-muted-foreground">
              Monitoring started {relativeTime(data.monitoring_since)}
            </p>
          ) : null}
        </div>
      </div>
    </Card>
  );
}
