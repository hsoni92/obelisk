"use client";

import { Button } from "@/components/ui/button";
import { Check, Copy } from "lucide-react";
import { useState } from "react";

export function CopyButton({
  value,
  label = "Copy",
}: {
  value: string;
  label?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await writeClipboard(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard denied */
    }
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="shrink-0"
      onClick={() => void copy()}
      aria-label={copied ? "Copied" : `${label} connection string`}
    >
      {copied ? (
        <Check className="size-3.5 text-success" />
      ) : (
        <Copy className="size-3.5" />
      )}
      <span className="sr-only">{copied ? "Copied" : label}</span>
    </Button>
  );
}

// The Clipboard API is only available in a secure context (HTTPS or localhost).
// A VPS admin UI on http://<ip> must fall back to a user-gesture textarea copy.
async function writeClipboard(value: string) {
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      /* permission denied or unavailable — try the legacy path */
    }
  }
  if (!copyWithTextarea(value)) {
    throw new Error("copy failed");
  }
}

function copyWithTextarea(value: string) {
  const area = document.createElement("textarea");
  area.value = value;
  area.setAttribute("readonly", "");
  area.style.position = "fixed";
  area.style.top = "0";
  area.style.left = "0";
  area.style.width = "1px";
  area.style.height = "1px";
  area.style.padding = "0";
  area.style.border = "none";
  area.style.outline = "none";
  area.style.boxShadow = "none";
  area.style.background = "transparent";
  // Stay inside the open dialog. A node on document.body is pulled back by the focus trap
  // before execCommand runs.
  const anchor =
    document.activeElement instanceof HTMLElement
      ? document.activeElement
      : document.body;
  anchor.insertAdjacentElement("afterend", area);
  area.focus();
  area.select();
  area.setSelectionRange(0, value.length);
  let ok = false;
  try {
    ok = document.execCommand("copy");
  } finally {
    area.remove();
  }
  return ok;
}
