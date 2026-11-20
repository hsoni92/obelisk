"use client";

import { cn } from "@/lib/utils";
import { Check, Copy } from "lucide-react";
import * as React from "react";

type CopyableIdProps = {
  value: string;
  /** Accessible name, e.g. "machine ID". */
  label?: string;
  /** "chip" = pill on light surfaces, "onDark" = for the green feature card, "block" = full-width readout. */
  variant?: "chip" | "onDark" | "block";
  className?: string;
};

const STYLES = {
  chip: "h-7 gap-1.5 rounded-full bg-muted px-2.5 text-[12px] text-foreground hover:bg-surface",
  onDark: "h-6 w-full justify-between gap-2 rounded-lg px-0 text-[12px] text-white hover:text-white/80",
  block: "w-full justify-between gap-3 rounded-xl bg-muted px-4 py-3 text-sm text-foreground hover:bg-surface",
} as const;

export function CopyableId({ value, label = "ID", variant = "chip", className }: CopyableIdProps) {
  const [copied, setCopied] = React.useState(false);

  async function copy(e: React.MouseEvent) {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard unavailable (insecure context) */
    }
  }

  return (
    <button
      type="button"
      onClick={(e) => void copy(e)}
      title={copied ? "Copied" : `Copy ${label}: ${value}`}
      aria-label={copied ? `${label} copied` : `Copy ${label}`}
      className={cn(
        "group inline-flex max-w-full items-center font-mono outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring",
        STYLES[variant],
        className,
      )}
    >
      <span className={cn("min-w-0 truncate", variant === "block" && "break-all whitespace-normal")}>{value}</span>
      <span
        className={cn(
          "shrink-0 transition-opacity",
          variant === "onDark" ? "opacity-70 group-hover:opacity-100" : "text-muted-foreground group-hover:text-foreground",
        )}
        aria-hidden
      >
        {copied ? (
          <Check className={cn("size-3.5", variant !== "onDark" && "text-success")} />
        ) : (
          <Copy className="size-3.5" />
        )}
      </span>
      <span role="status" className="sr-only">{copied ? "Copied" : ""}</span>
    </button>
  );
}
