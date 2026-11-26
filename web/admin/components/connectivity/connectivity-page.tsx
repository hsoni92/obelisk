"use client";

import { GatewayPeerView } from "@/components/connectivity/gateway-peer-view";
import { TailscaleFederationPanel } from "@/components/platform/tailscale-federation-panel";

/** Tailscale join and gateway role first (setup), then the live peer view on a gateway. */
export function ConnectivityPage() {
  return (
    <div className="max-w-3xl space-y-6">
      <TailscaleFederationPanel />
      <GatewayPeerView />
    </div>
  );
}
