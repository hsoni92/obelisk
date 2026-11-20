"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
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
import { FederationPeersPanel } from "@/components/platform/federation-peers-panel";
import { GatewayAdvertiseUrlCopy } from "@/components/platform/gateway-advertise-url-copy";
import { ExternalLink, Network } from "lucide-react";
import * as React from "react";

const TAILSCALE_AUTH_KEYS_URL =
  "https://login.tailscale.com/admin/settings/keys";
const TAILSCALE_AUTH_KEYS_DOCS_URL = "https://tailscale.com/kb/1247/auth-keys";

function TailscaleExternalLink({
  href,
  children,
}: {
  href: string;
  children: React.ReactNode;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-0.5 text-broadcast underline"
    >
      {children}
      <ExternalLink className="size-3" aria-hidden />
    </a>
  );
}

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
  const profile = data?.deployment_profile ?? "vps";
  const vpsBlocksTailscaleInConfig =
    profile === "vps" && !isGateway;

  async function onConnect() {
    const key = authKey.trim();
    if (!key && !connected) {
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
    <section aria-labelledby="tailscale-heading">
      <Card className="divide-y divide-border/60 overflow-hidden">
        {/* Header: identity + the one status that matters */}
        <div className="flex items-start justify-between gap-4 p-6">
          <div className="flex min-w-0 items-start gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
              <Network className="size-4" aria-hidden />
            </span>
            <div className="min-w-0">
              <h2 id="tailscale-heading" className="text-page-title">
                Tailscale &amp; gateway
              </h2>
              <p className="mt-0.5 text-body text-muted-foreground">
                Join your tailnet and optionally make this box the public front
                for home Obelisks.
              </p>
            </div>
          </div>
          <Badge variant={connected ? "success" : "warning"} className="shrink-0">
            {connected ? "Connected" : "Not connected"}
          </Badge>
        </div>

        {/* Connection */}
        <div className="space-y-4 p-6">
          {error && !data ? (
            <p className="text-sm text-destructive">{rtkErrorMessage(error)}</p>
          ) : null}

          <div className="flex flex-wrap items-center gap-2">
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
            <Badge variant="outline">profile {profile}</Badge>
          </div>

          {vpsBlocksTailscaleInConfig ? (
            <p className="text-sm text-muted-foreground">
              Standalone VPS profile keeps Tailscale off unless you enable{" "}
              <span className="font-medium text-foreground">Gateway appliance</span>{" "}
              below, or set{" "}
              <span className="font-mono text-[12px]">deployment.profile: local</span>{" "}
              for a home machine (e.g. dev/obelisk.yaml).
            </p>
          ) : null}

          {st?.dns_name ? (
            <p className="rounded-xl bg-muted px-4 py-3 font-mono text-[13px] text-muted-foreground">
              {st.dns_name}
              {st.ips?.length ? ` · ${st.ips.join(", ")}` : ""}
              {st.tailnet ? ` · ${st.tailnet}` : ""}
            </p>
          ) : null}

          {st?.auth_url ? (
            <p className="text-sm">
              <TailscaleExternalLink href={st.auth_url}>
                Open Tailscale login
              </TailscaleExternalLink>
            </p>
          ) : null}

          {st?.error ? (
            <p className="text-sm text-muted-foreground">{st.error}</p>
          ) : null}

          <div className="space-y-2">
            <Label htmlFor="tailscale-auth-key">Auth key</Label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input
                id="tailscale-auth-key"
                type="password"
                autoComplete="off"
                className="min-w-0 flex-1 rounded-lg border border-border bg-background px-3 py-2 font-mono text-sm"
                placeholder="tskey-auth-…"
                value={authKey}
                onChange={(e) => setAuthKey(e.target.value)}
              />
              <Button
                type="button"
                disabled={connectState.isLoading || isLoading}
                onClick={() => void onConnect()}
              >
                Connect tailnet
              </Button>
            </div>
            <p className="text-metadata leading-relaxed text-muted-foreground">
              One-time or reusable key, never stored in obelisk.yaml. Enable
              pre-approval for headless VPS join.{" "}
              <TailscaleExternalLink href={TAILSCALE_AUTH_KEYS_URL}>
                Generate a key
              </TailscaleExternalLink>
              {" · "}
              <TailscaleExternalLink href={TAILSCALE_AUTH_KEYS_DOCS_URL}>
                How auth keys work
              </TailscaleExternalLink>
            </p>
          </div>
        </div>

        {/* Gateway */}
        <div className="flex items-center justify-between gap-4 p-6">
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

        {isGateway ? (
          <>
            <div className="px-6 pb-2">
              <GatewayAdvertiseUrlCopy
                endpoint={data?.gateway_advertise_endpoint}
                tailnetEndpoint={data?.gateway_advertise_endpoint_tailnet}
              />
            </div>
            <FederationPeersPanel isGateway />
          </>
        ) : profile === "local" ? (
          <FederationPeersPanel isGateway={false} />
        ) : (
          <p className="px-6 pb-6 text-sm text-muted-foreground">
            Enable <span className="font-medium">Gateway appliance</span> on this
            VPS to list remote peers and show home services as federated rows.
          </p>
        )}
      </Card>
    </section>
  );
}
