"use client";

import { MailPanel } from "@/components/exposure/mail-panel";
import { OpenRouterPanel } from "@/components/exposure/openrouter-panel";
import { PageHeader } from "@/components/layout/page-header";
import { useState } from "react";

export default function ExposurePage() {
  const [, setBusy] = useState<string | null>(null);

  return (
    <>
      <PageHeader section="exposure" />

      <div className="divide-y divide-border rounded-xl bg-card shadow-rb-2">
        <OpenRouterPanel onBusy={setBusy} />
        <MailPanel onBusy={setBusy} />
      </div>
    </>
  );
}
