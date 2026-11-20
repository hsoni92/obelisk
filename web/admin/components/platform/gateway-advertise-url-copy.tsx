"use client";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { toast } from "@/lib/toast";
import { Copy } from "lucide-react";

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Gateway advertise URL copied");
  } catch {
    toast.error("Could not copy to clipboard");
  }
}

export function GatewayAdvertiseUrlCopy({
  endpoint,
}: {
  endpoint: string | undefined;
}) {
  if (!endpoint) {
    return <AdvertiseUrlMissingHint />;
  }

  return (
    <div className="space-y-2 rounded-xl border border-broadcast/30 bg-broadcast-tint/40 p-4">
      <Label className="text-foreground">Gateway advertise URL</Label>
      <p className="text-metadata text-muted-foreground">
        Paste this on each local Obelisk under Platform → Register with
        gateway.
      </p>
      <div className="flex flex-col gap-2 sm:flex-row">
        <code className="min-w-0 flex-1 rounded-lg border border-border bg-background px-3 py-2.5 font-mono text-[12px] leading-snug break-all">
          {endpoint}
        </code>
        <Button
          type="button"
          variant="default"
          className="shrink-0"
          onClick={() => void copyText(endpoint)}
        >
          <Copy className="size-4" aria-hidden />
          Copy URL
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