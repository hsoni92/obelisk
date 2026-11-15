"use client";

import { MailPanel } from "@/components/exposure/mail-panel";
import { OpenRouterPanel } from "@/components/exposure/openrouter-panel";
import { PageHeader } from "@/components/layout/page-header";
import { useState } from "react";

export default function ExposurePage() {
  const [, setBusy] = useState<string | null>(null);
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

      <div className="divide-y divide-border rounded-xl bg-card shadow-rb-2">
        <OpenRouterPanel onBusy={setBusy} onError={setActionError} />
        <MailPanel onBusy={setBusy} onError={setActionError} />
      </div>
    </>
  );
}
