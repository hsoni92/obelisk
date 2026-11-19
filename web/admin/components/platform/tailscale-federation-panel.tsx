"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CardHeading } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/lib/toast";
import {
  rtkErrorMessage,
  tailscalePollOptions,
  useConnectTailscaleMutation,
  useGetTailscaleStatusQuery,
  usePatchDeploymentMutation,
} from "@/lib/store/obelisk-api";
import { ExternalLink, Network } from "lucide-react";
import * as React from "react";

export function TailscaleFederationPanel() {
  const { data, isLoading, error } = useGetTailscaleStatusQuery(
    undefined,
    tailscalePollOptions,
  );
  const [connect, connectState] = useConnectTailscaleMutation();
  const [patchDeployment, patchState] = usePatchDeploymentMutation();
  const [authKey, setAuthKey] = React.useState("");

  const st = data?.status;
  const connected = Boolean(st?.connected);
  const isGateway = data?.deployment_role === "gateway";

  async function onConnect() {
    const key = authKey.trim();
    if (!key) {
      toast.error("Paste a Tailscale auth key first");
      return;
    }
    try {
      await connect({
        auth_key: key,
        enable_in_config: true,
        gateway_role: isGateway ? true : undefined,
      }).unwrap();
      setAuthKey("");
      toast.success("Tailscale connect requested");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  async function onGatewayToggle(checked: boolean) {
    try {
      await patchDeployment({
        role: checked ? "gateway" : "local",
        tailscale_enabled: checked ? true : undefined,
      }).unwrap();
      toast.success(checked ? "Gateway role enabled" : "Local role set");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  return (
    <section id="tailscale" aria-labelledby="tailscale-heading" className="scroll-mt-20">
      <CardHeading
        icon={Network}
        title="Tailscale & gateway"
        description="Join your tailnet from the console. Use an auth key from the Tailscale admin (Settings → Keys). Gateway role turns this box into the public front for home Obelisks."
      />
      <div className="mt-4 space-y-4 rounded-xl border border-border bg-card p-4 shadow-rb-2">
        {error && !data ? (
          <p className="text-sm text-destructive">{rtkErrorMessage(error)}</p>
        ) : null}

        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={connected ? "success" : "warning"}>
            {connected ? "Connected" : "Not connected"}
          </Badge>
          {data?.desired_enabled ? (
            <Badge variant="info">enabled in config</Badge>
          ) : (
            <Badge variant="outline">not in config</Badge>
          )}
          {isGateway ? (
            <Badge variant="warning">gateway</Badge>
          ) : (
            <Badge variant="outline">local role</Badge>
          )}
        </div>

        {st?.dns_name ? (
          <p className="font-mono text-[13px] text-muted-foreground">
            {st.dns_name}
            {st.ips?.length ? ` · ${st.ips.join(", ")}` : ""}
            {st.tailnet ? ` · ${st.tailnet}` : ""}
          </p>
        ) : null}

        {st?.auth_url ? (
          <p className="text-sm">
            <a
              href={st.auth_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-broadcast underline"
            >
              Open Tailscale login
              <ExternalLink className="size-3.5" />
            </a>
          </p>
        ) : null}

        {st?.error ? (
          <p className="text-sm text-muted-foreground">{st.error}</p>
        ) : null}

        <div className="space-y-1">
          <Label htmlFor="tailscale-auth-key">Auth key</Label>
          <p className="text-[11px] text-muted-foreground">
            One-time or reusable key; not stored in obelisk.yaml
          </p>
          <input
            id="tailscale-auth-key"
            type="password"
            autoComplete="off"
            className="w-full rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm"
            placeholder="tskey-auth-…"
            value={authKey}
            onChange={(e) => setAuthKey(e.target.value)}
          />
        </div>

        <Button
          type="button"
          disabled={connectState.isLoading || isLoading}
          onClick={() => void onConnect()}
        >
          Connect tailnet
        </Button>

        <div className="flex items-center justify-between gap-3 border-t border-border pt-4">
          <div>
            <p className="text-sm font-medium">Gateway appliance</p>
            <p className="text-metadata text-muted-foreground">
              Enables mesh backhaul and peer registration on this VPS.
            </p>
          </div>
          <Switch
            checked={isGateway}
            disabled={patchState.isLoading}
            onCheckedChange={(v) => void onGatewayToggle(v)}
            aria-label="Gateway role"
          />
        </div>
      </div>
    </section>
  );
}

// framer-motion not needed — remove erroneous motion tag