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
import { Label } from "@/components/ui/label";
import { toast } from "@/lib/toast";
import { relativeTime } from "@/lib/overview-series";
import {
  federationPollOptions,
  rtkErrorMessage,
  useDeleteFederationPeerMutation,
  useFederationAdvertiseNowMutation,
  useFederationUnregisterMutation,
  useGetFederationQuery,
  usePatchFederationMutation,
} from "@/lib/store/obelisk-api";
import type { FederationPeerRecord } from "@/lib/types";
import { RefreshCw, Server } from "lucide-react";
import * as React from "react";

function instanceCount(peer: FederationPeerRecord): number {
  if (!peer.instances) return 0;
  return Object.values(peer.instances).filter((i) => i.enabled).length;
}

function sortPeers(list: FederationPeerRecord[]): FederationPeerRecord[] {
  return [...list].sort((a, b) => {
    const ta = a.last_seen ? Date.parse(a.last_seen) : 0;
    const tb = b.last_seen ? Date.parse(b.last_seen) : 0;
    if (tb !== ta) return tb - ta;
    return a.id.localeCompare(b.id);
  });
}

export function FederationPeersPanel({
  isGateway,
}: {
  isGateway: boolean;
}) {
  const { data, isLoading, error } = useGetFederationQuery(
    undefined,
    federationPollOptions,
  );
  const [patchFederation, patchState] = usePatchFederationMutation();
  const [advertiseNow, advertiseState] = useFederationAdvertiseNowMutation();
  const [unregister, unregisterState] = useFederationUnregisterMutation();
  const [deletePeer, deletePeerState] = useDeleteFederationPeerMutation();
  const [gatewayURL, setGatewayURL] = React.useState("");
  const [peerToRemove, setPeerToRemove] = React.useState<string | null>(null);
  const [confirmUnregister, setConfirmUnregister] = React.useState(false);

  React.useEffect(() => {
    if (data?.advertise_url != null) {
      setGatewayURL(data.advertise_url);
    }
  }, [data?.advertise_url]);

  const peers = React.useMemo(
    () => sortPeers(data?.peers ?? []),
    [data?.peers],
  );
  const advertise = data?.advertise_status;

  async function onSaveGatewayURL() {
    const url = gatewayURL.trim();
    if (!url) {
      toast.error("Paste the gateway advertise URL");
      return;
    }
    try {
      await patchFederation({ advertise_url: url }).unwrap();
      toast.success("Gateway URL saved — advertising every ~30s");
      void advertiseNow().unwrap().catch(() => undefined);
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  async function onAdvertiseNow() {
    try {
      await advertiseNow().unwrap();
      toast.success("Catalog sent to gateway");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  async function onConfirmRemovePeer() {
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

  async function onConfirmUnregister() {
    try {
      const res = await unregister().unwrap();
      setGatewayURL("");
      setConfirmUnregister(false);
      toast.success("Unregistered from gateway");
      if (res.gateway_warning) {
        toast.warning(res.gateway_warning);
      }
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  if (error && !data) {
    return (
      <p className="text-sm text-destructive px-6 pb-6">
        {rtkErrorMessage(error)}
      </p>
    );
  }

  if (isGateway) {
    return (
      <div className="space-y-4 p-6">
        <div className="flex items-start gap-3">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-broadcast-tint text-foreground">
            <Server className="size-4" aria-hidden />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium">Remote peers</p>
            <p className="text-metadata text-muted-foreground">
              Home Obelisks register here and show up as federated services on
              this gateway.
            </p>
          </div>
          <Badge variant={peers.length ? "success" : "outline"}>
            {peers.length ? `${peers.length} peer${peers.length === 1 ? "" : "s"}` : "No peers yet"}
          </Badge>
        </div>

        {peers.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border px-4 py-3 text-sm text-muted-foreground">
            Waiting for a local appliance. Copy the gateway advertise URL above,
            then on the home box: Connectivity → Register with gateway.
          </p>
        ) : (
          <ul className="divide-y divide-border/60 rounded-xl border border-border overflow-hidden">
            {peers.map((peer) => (
              <li
                key={peer.id}
                className="flex flex-wrap items-center justify-between gap-2 bg-card px-4 py-3 text-sm"
              >
                <div className="min-w-0">
                  <p className="font-medium truncate">{peer.id}</p>
                  <p className="font-mono text-[11px] text-muted-foreground truncate">
                    {peer.tailnet_host || "—"}
                    {peer.machine_id ? ` · ${peer.machine_id}` : ""}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="outline">
                    {instanceCount(peer)} service
                    {instanceCount(peer) === 1 ? "" : "s"}
                  </Badge>
                  <Badge variant={peer.reachable ? "success" : "warning"}>
                    {peer.reachable ? "reachable" : "stale"}
                  </Badge>
                  {peer.last_seen ? (
                    <span className="text-metadata text-muted-foreground">
                      seen {relativeTime(peer.last_seen)}
                    </span>
                  ) : null}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="text-destructive border-destructive/30 hover:bg-destructive/10"
                    disabled={deletePeerState.isLoading}
                    onClick={() => setPeerToRemove(peer.id)}
                  >
                    Remove
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
        {isLoading ? (
          <p className="text-metadata text-muted-foreground">Refreshing…</p>
        ) : null}
        <AlertDialog
          open={peerToRemove != null}
          onOpenChange={(open) => !open && setPeerToRemove(null)}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Remove peer from gateway?</AlertDialogTitle>
              <AlertDialogDescription>
                Federated services for{" "}
                <span className="font-mono">{peerToRemove}</span> disappear from
                this gateway until the peer registers again.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                onClick={() => void onConfirmRemovePeer()}
              >
                Remove peer
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    );
  }

  return (
    <div className="space-y-4 p-6">
      <div>
        <p className="text-sm font-medium">Register with gateway</p>
        <p className="text-metadata text-muted-foreground">
          Point this appliance at your gateway&apos;s advertise URL. Enabled
          services appear on the gateway under federated rows.
        </p>
      </div>

      {!data ? (
        <p className="text-metadata text-muted-foreground">Checking tailnet…</p>
      ) : data.tailnet_host ? (
        <p className="text-metadata text-muted-foreground">
          Tailnet dial host:{" "}
          <span className="font-mono text-foreground">{data.tailnet_host}</span>
        </p>
      ) : (
        <p className="text-sm text-amber-700 dark:text-amber-400">
          Connect Tailscale first so the gateway can reach this box.
        </p>
      )}

      <div className="space-y-2">
        <Label htmlFor="gateway-advertise-url">Gateway advertise URL</Label>
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            id="gateway-advertise-url"
            type="url"
            autoComplete="off"
            className="min-w-0 flex-1 rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm"
            placeholder="http://203.0.113.1:9080/api/v1/peers/advertise"
            value={gatewayURL}
            onChange={(e) => setGatewayURL(e.target.value)}
          />
          <Button
            type="button"
            disabled={patchState.isLoading}
            onClick={() => void onSaveGatewayURL()}
          >
            Save & register
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!data?.advertise_configured || advertiseState.isLoading}
          onClick={() => void onAdvertiseNow()}
        >
          <RefreshCw className="size-3.5" aria-hidden />
          Push catalog now
        </Button>
        {advertise ? (
          <Badge variant={advertise.ok ? "success" : "warning"}>
            {advertise.ok ? "Last push OK" : "Last push failed"}
          </Badge>
        ) : (
          <Badge variant="outline">No push yet</Badge>
        )}
        {advertise?.last_attempt_at ? (
          <span className="text-metadata text-muted-foreground">
            {relativeTime(advertise.last_attempt_at)}
          </span>
        ) : null}
      </div>

      {advertise?.last_error && !advertise.ok ? (
        <p className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {advertise.last_error}
        </p>
      ) : null}

      {data?.advertise_configured ? (
        <div className="border-t border-border/60 pt-4">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="text-destructive border-destructive/30 hover:bg-destructive/10"
            disabled={unregisterState.isLoading}
            onClick={() => setConfirmUnregister(true)}
          >
            Unregister from gateway
          </Button>
        </div>
      ) : null}

      <AlertDialog open={confirmUnregister} onOpenChange={setConfirmUnregister}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unregister from gateway?</AlertDialogTitle>
            <AlertDialogDescription>
              Stops advertising, removes backhaul routes on this box, and asks
              the gateway to drop this peer. Re-register anytime with a new save.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => void onConfirmUnregister()}
            >
              Unregister
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}