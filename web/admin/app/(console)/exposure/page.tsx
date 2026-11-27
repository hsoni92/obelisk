"use client";

import { MCPPanel } from "@/components/exposure/mcp-panel";
import { MailPanel } from "@/components/exposure/mail-panel";
import { OpenRouterPanel } from "@/components/exposure/openrouter-panel";
import { PageHeader } from "@/components/layout/page-header";
import { useState } from "react";

export default function ExposurePage() {
  const [, setBusy] = useState<string | null>(null);

  return (
    <>
      <PageHeader section="exposure" />

      <div className="grid gap-4 lg:grid-cols-2">
        <OpenRouterPanel onBusy={setBusy} />
        <MailPanel onBusy={setBusy} />
        <MCPPanel onBusy={setBusy} />
      </div>
    </>
  );
}
