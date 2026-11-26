"use client";

import type { ConnectivityPeer, ConnectivityState } from "@/lib/types";
import { STATE_DOT_COLOR } from "@/components/connectivity/connectivity-format";

const WIDTH = 640;
const ROW = 30;
const PAD = 16;
const HUB_X = 72;
const NODE_X = 380;
const LABEL_MAX = 26;

type PeerFlowDiagramProps = {
  peers: ConnectivityPeer[];
  maxRows?: number;
};

function shortName(id: string) {
  return id.length > LABEL_MAX ? `${id.slice(0, LABEL_MAX - 1)}…` : id;
}

/**
 * Decorative flow of peers into this gateway. Dots move toward the gateway for
 * peers that are heartbeating. The count in the hub is the number of connected
 * peers. Screen readers get the same information from the list beside it.
 */
export function PeerFlowDiagram({ peers, maxRows = 8 }: PeerFlowDiagramProps) {
  const shown = peers.slice(0, maxRows);
  const connected = peers.filter((p) => p.state === "connected").length;
  const height = Math.max(120, PAD * 2 + shown.length * ROW);
  const hubY = height / 2;

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${height}`}
      className="h-auto w-full"
      aria-hidden="true"
      focusable="false"
    >
      {shown.map((peer, i) => {
        const y = PAD + ROW / 2 + i * ROW;
        const state: ConnectivityState = peer.state;
        const live = state === "connected" || state === "degraded";
        const color = STATE_DOT_COLOR[state];
        const d = `M ${NODE_X} ${y} C ${NODE_X - 140} ${y}, ${HUB_X + 70} ${hubY}, ${HUB_X + 26} ${hubY}`;
        return (
          <g key={peer.id}>
            <path
              d={d}
              fill="none"
              stroke={color}
              strokeWidth={1.5}
              strokeLinecap="round"
              strokeDasharray="2 6"
              opacity={live ? 0.9 : 0.35}
              className={live ? "conn-flow" : undefined}
              data-speed={state === "degraded" ? "slow" : undefined}
            />
            <circle cx={NODE_X} cy={y} r={5} fill={color} />
            <text
              x={NODE_X + 14}
              y={y + 4}
              fontSize={12}
              fill="currentColor"
              className="text-foreground"
            >
              {shortName(peer.id)}
            </text>
          </g>
        );
      })}
      <circle cx={HUB_X} cy={hubY} r={24} fill="var(--broadcast)" />
      <text
        x={HUB_X}
        y={hubY + 5}
        textAnchor="middle"
        fontSize={15}
        fontWeight={600}
        fill="var(--broadcast-foreground)"
      >
        {connected}
      </text>
      <text
        x={HUB_X}
        y={hubY + 42}
        textAnchor="middle"
        fontSize={11}
        fill="currentColor"
        className="text-muted-foreground"
      >
        connected
      </text>
    </svg>
  );
}
