"use client";

import { HeatStrip } from "@/components/connectivity/heat-strip";
import { PeerFlowScene } from "@/components/connectivity/peer-flow-scene";
import { aggregateStrip, availability } from "@/components/connectivity/peer-stats";
import { Badge } from "@/components/ui/badge";
import { Card, CardHeading } from "@/components/ui/card";
import { connectivityPollOptions, useGetConnectivityQuery } from "@/lib/store/obelisk-api";
import { ArrowUpRight, Waypoints } from "lucide-react";
import Link from "next/link";
import { useMemo } from "react";

/**
 * Overview card for the gateway peer flow. Renders nothing until the role is
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
      <div className="grid items-center gap-x-6 gap-y-3 p-3 sm:grid-cols-[17.5rem_minmax(0,1fr)] sm:p-4">
        <div
          role="img"
          aria-label={`${summary}. ${degraded} degraded, ${offline} offline.`}
          className="mx-auto w-full max-w-[17.5rem]"
        >
          <PeerFlowScene peers={peers} connected={connected} />
        </div>

        <div className="space-y-3">
          <div className="flex items-baseline gap-2">
            <p className="text-3xl font-light leading-none tabular-nums text-foreground">
              {avail === null ? "—" : `${avail >= 99.95 ? 100 : avail.toFixed(1)}%`}
            </p>
            <p className="text-metadata text-muted-foreground">available, {data.window_hours} h</p>
          </div>

          {peers.length > 0 ? (
            <div className="space-y-1.5">
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
        </div>
      </div>
    </Card>
  );
}
