"use client";

import { MailPanel } from "@/components/exposure/mail-panel";
import { OpenRouterPanel } from "@/components/exposure/openrouter-panel";
import { PageHeader } from "@/components/layout/page-header";
import { useState } from "react";

export default function ExposurePage() {
  const [busy, setBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  return (
    <>
      <PageHeader section="exposure" />

      {actionError && (
        <p
          className="text-xs font-medium text-broadcast bg-broadcast-tint rounded-full px-3 py-1 mb-3 inline-block"
          role="alert"
        >
          {actionError}
        </p>
      )}

      <div className="space-y-4">
        <p className="text-body text-muted-foreground max-w-2xl">
          VPS: public HTTP via Kong. Funnel only when local + Tailscale.
        </p>
        <OpenRouterPanel onBusy={setBusy} onError={setActionError} />
        <MailPanel onBusy={setBusy} onError={setActionError} />
      </div>
    </>
  );
}