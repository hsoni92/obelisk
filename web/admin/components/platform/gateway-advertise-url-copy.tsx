"use client";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { copyToClipboard } from "@/lib/copy-to-clipboard";
import { toast } from "@/lib/toast";
import { Copy } from "lucide-react";

async function copyText(text: string, label: string) {
  try {
    await copyToClipboard(text);
    toast.success(`${label} URL copied`);
  } catch {
    toast.error("Could not copy to clipboard");
  }
}

export function GatewayAdvertiseUrlCopy({
  endpoint,
  tailnetEndpoint,
}: {
  endpoint: string | undefined;
  tailnetEndpoint?: string;
}) {
  if (!endpoint) {
    return <AdvertiseUrlMissingHint />;
  }

  return (
    <div className="space-y-3 rounded-xl border border-broadcast/30 bg-broadcast-tint/40 p-4">
      <Label className="text-foreground">Gateway advertise URL</Label>
      <p className="text-metadata text-muted-foreground">
        Paste this on each local Obelisk under Connectivity → Register with
        gateway. Prefer the tailnet URL if the public one times out.
      </p>
      <UrlRow label="Public" url={endpoint} />
      {tailnetEndpoint ? (
        <UrlRow label="Tailnet" url={tailnetEndpoint} />
      ) : null}
    </div>
  );
}

function UrlRow({ label, url }: { label: string; url: string }) {
  return (
    <div className="space-y-1">
      <span className="text-eyebrow text-muted-foreground">{label}</span>
      <div className="flex flex-col gap-2 sm:flex-row">
        <code className="min-w-0 flex-1 rounded-lg border border-border bg-background px-3 py-2.5 font-mono text-[12px] leading-snug break-all">
          {url}
        </code>
        <Button
          type="button"
          variant={label === "Tailnet" ? "outline" : "default"}
          className="shrink-0"
          onClick={() => void copyText(url, label)}
        >
          <Copy className="size-4" aria-hidden />
          Copy
        </Button>
      </div>
    </div>
  );
}

function AdvertiseUrlMissingHint() {
  return (
    <p className="rounded-xl border border-dashed border-border px-4 py-3 text-sm text-muted-foreground">
      Connect Tailscale or set a public host so this gateway can show an
      advertise URL for locals.
    </p>
  );
}