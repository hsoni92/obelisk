"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { PeerFlowScene } from "@/components/connectivity/peer-flow-scene";
import { availability } from "@/components/connectivity/peer-stats";
import { PeerStrip } from "@/components/connectivity/peer-strip";
import {
  STATE_LABEL,
  formatSpan,
} from "@/components/connectivity/connectivity-format";
import { relativeTime } from "@/lib/overview-series";
import {
  connectivityPollOptions,
  rtkErrorMessage,
  useDeleteFederationPeerMutation,
  useGetConnectivityQuery,
} from "@/lib/store/obelisk-api";
import { toast } from "@/lib/toast";
import type { ConnectivityPeer } from "@/lib/types";
import { Network } from "lucide-react";
import * as React from "react";

/** One sentence per peer: how long it has been in its current state. */
function describeState(peer: ConnectivityPeer): string {
  const prefix = peer.state_since_capped ? "≥ " : "";
  switch (peer.state) {
    case "connected":
      return `Up for ${prefix}${formatSpan(peer.observed_seconds)}`;
    case "offline":
      return `Down for ${prefix}${formatSpan(peer.observed_seconds)}`;
    case "degraded":
      return `Degraded for ${prefix}${formatSpan(peer.observed_seconds)}`;
    case "never":
      return "Waiting for its first heartbeat";
    default:
      return STATE_LABEL[peer.state];
  }
}

function probeLine(peer: ConnectivityPeer): string {
  if (!peer.probe_known) return "Reachability check pending";
  const checked = peer.probe_checked_at
    ? ` · checked ${relativeTime(peer.probe_checked_at)}`
    : "";
  if (peer.probe_reachable) return `Reachable${checked}`;
  return `Not reachable${checked}`;
}

function stateBadge(peer: ConnectivityPeer) {
  switch (peer.state) {
    case "connected":
      return <Badge variant="success">{STATE_LABEL.connected}</Badge>;
    case "degraded":
      return <Badge variant="warning">{STATE_LABEL.degraded}</Badge>;
    case "offline":
      return (
        <Badge className="bg-destructive-tint text-destructive">
          {STATE_LABEL.offline}
        </Badge>
      );
    default:
      return <Badge variant="outline">{STATE_LABEL[peer.state]}</Badge>;
  }
}

function PeerRow({
  peer,
  windowStart,
  binMinutes,
  onRemove,
  removing,
}: {
  peer: ConnectivityPeer;
  windowStart?: string;
  binMinutes: number;
  onRemove: (id: string) => void;
  removing: boolean;
}) {
  return (
    <li className="space-y-3 bg-card px-4 py-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{peer.id}</p>
          <p className="truncate font-mono text-[11px] text-muted-foreground">
            {peer.tailnet_host || "no tailnet address"}
            {peer.machine_id ? ` · ${peer.machine_id}` : ""}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {stateBadge(peer)}
          <span className="text-metadata text-muted-foreground">
            {describeState(peer)}
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="text-destructive border-destructive/30 hover:bg-destructive/10"
            disabled={removing}
            onClick={() => onRemove(peer.id)}
          >
            Remove
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-metadata text-muted-foreground">
        <span>
          {peer.last_seen ? `Heartbeat ${relativeTime(peer.last_seen)}` : "No heartbeat yet"}
        </span>
        <span title={peer.probe_error || undefined} className="truncate">
          {probeLine(peer)}
        </span>
        {peer.duplicate_machine_id ? (
          <Badge variant="warning">Duplicate machine ID</Badge>
        ) : null}
      </div>

      <PeerStrip
        peerId={peer.id}
        bins={peer.strip}
        windowStart={windowStart}
        binMinutes={binMinutes}
      />
    </li>
  );
}

function StripLegend() {
  const items: { label: string; color: string }[] = [
    { label: "Connected", color: "var(--success)" },
    { label: "Degraded", color: "var(--warning)" },
    { label: "Offline", color: "var(--destructive)" },
    { label: "No data", color: "var(--muted)" },
  ];
  return (
    <ul className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-muted-foreground">
      {items.map((item) => (
        <li key={item.label} className="flex items-center gap-1.5">
          <span
            className="inline-block size-2.5 rounded-sm"
            style={{ backgroundColor: item.color }}
            aria-hidden="true"
          />
          {item.label}
        </li>
      ))}
    </ul>
  );
}

export function GatewayPeerView() {
  const { data, error } = useGetConnectivityQuery(
    undefined,
    connectivityPollOptions,
  );
  const [deletePeer, deleteState] = useDeleteFederationPeerMutation();
  const [peerToRemove, setPeerToRemove] = React.useState<string | null>(null);

  async function onConfirmRemove() {
    const id = peerToRemove;
    if (!id) return;
    try {
      await deletePeer(id).unwrap();
      toast.success(`Removed peer ${id}`);
      setPeerToRemove(null);
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  if (error && !data) {
    return (
      <Card className="p-6">
        <p className="text-sm text-destructive">{rtkErrorMessage(error)}</p>
      </Card>
    );
  }
  // Render nothing until the role is known. A local host never shows this card.
  if (!data) return null;
  if (!data.peer_view_enabled) return null;

  const peers = data.peers;
  const avail = availability(peers);
  const counts = data.counts;
  return (
    <section aria-labelledby="peers-heading">
      <Card className="space-y-5 p-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-broadcast-tint text-foreground">
              <Network className="size-4" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 id="peers-heading" className="text-page-title">
                Peers
              </h2>
              <p className="mt-0.5 text-body text-muted-foreground">
                Home Obelisks that send heartbeats to this gateway, and that this
                gateway can reach.
              </p>
            </div>
          </div>
          <Badge variant={peers.length ? "success" : "outline"} className="shrink-0">
            {peers.length ? `${peers.length} peer${peers.length === 1 ? "" : "s"}` : "No peers"}
          </Badge>
        </div>

        {peers.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border px-4 py-3 text-sm text-muted-foreground">
            Waiting for a local appliance. Copy the gateway advertise URL from the
            Tailscale card above, then on the home box: Connectivity → Register
            with gateway.
          </p>
        ) : (
          <>
            <div className="grid items-center gap-x-6 gap-y-3 sm:grid-cols-[17.5rem_minmax(0,1fr)]">
              <div
                role="img"
                aria-label={`${counts.connected ?? 0} of ${peers.length} peers connected`}
                className="mx-auto w-full max-w-[17.5rem]"
              >
                <PeerFlowScene peers={peers} connected={counts.connected ?? 0} />
              </div>
              <div className="space-y-3">
                <div className="flex items-baseline gap-2">
                  <p className="text-3xl font-light leading-none tabular-nums text-foreground">
                    {avail === null ? "—" : `${avail >= 99.95 ? 100 : avail.toFixed(1)}%`}
                  </p>
                  <p className="text-metadata text-muted-foreground">
                    available, {data.window_hours} h
                  </p>
                </div>
                <ul className="flex flex-wrap gap-x-5 gap-y-1 text-metadata">
                  <li className="text-muted-foreground">
                    <span className="font-medium tabular-nums" style={{ color: "var(--success)" }}>
                      {counts.connected ?? 0}
                    </span>{" "}
                    connected
                  </li>
                  <li className="text-muted-foreground">
                    <span className="font-medium tabular-nums" style={{ color: "var(--warning)" }}>
                      {counts.degraded ?? 0}
                    </span>{" "}
                    degraded
                  </li>
                  <li className="text-muted-foreground">
                    <span className="font-medium tabular-nums" style={{ color: "var(--destructive)" }}>
                      {counts.offline ?? 0}
                    </span>{" "}
                    offline
                  </li>
                  {counts.never ? (
                    <li className="text-muted-foreground">
                      <span className="font-medium tabular-nums">{counts.never}</span> waiting
                    </li>
                  ) : null}
                </ul>
              </div>
            </div>
            <ul className="divide-y divide-border/60 overflow-hidden rounded-xl border border-border">
              {peers.map((peer) => (
                <PeerRow
                  key={peer.id}
                  peer={peer}
                  windowStart={data.window_start}
                  binMinutes={data.bin_minutes}
                  onRemove={setPeerToRemove}
                  removing={deleteState.isLoading}
                />
              ))}
            </ul>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <StripLegend />
              <p className="text-[11px] text-muted-foreground">
                Last {data.window_hours} h in {data.bin_minutes}-minute bins
              </p>
            </div>
          </>
        )}

        <p className="border-t border-border/60 pt-4 text-[11px] leading-relaxed text-muted-foreground">
          {data.monitoring_since
            ? `Monitoring since ${new Date(data.monitoring_since).toLocaleString()}. `
            : "Not monitoring yet. "}
          {data.gateway_alive_at
            ? `Gateway last alive ${relativeTime(data.gateway_alive_at)}. `
            : ""}
          Uptime counts only while this gateway was running. Heartbeats arrive about
          every 30 seconds; a peer is offline after {Math.round(data.heartbeat_stale_seconds / 60)} minutes
          without one. Reachability is a TCP check every {data.probe_interval_seconds} seconds.
        </p>
      </Card>

      <AlertDialog
        open={peerToRemove != null}
        onOpenChange={(open) => !open && setPeerToRemove(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove peer from gateway?</AlertDialogTitle>
            <AlertDialogDescription>
              Federated services for <span className="font-mono">{peerToRemove}</span>{" "}
              disappear from this gateway until the peer registers again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => void onConfirmRemove()}
            >
              Remove peer
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
