"use client";

import {
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastSurface,
  ToastTitle,
  ToastViewport,
} from "@/components/ui/toast";
import {
  dismiss,
  getToasts,
  getToastsServerSnapshot,
  remove,
  subscribe,
  type ToastRecord,
  type ToastVariant,
} from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  AlertTriangle,
  Check,
  CheckCircle2,
  ChevronDown,
  Copy,
  Info,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import { useState, useSyncExternalStore } from "react";

const VARIANT_STYLE: Record<
  ToastVariant,
  { icon: LucideIcon; tile: string; bar: string; title: string }
> = {
  default: { icon: Info, tile: "bg-muted text-muted-foreground", bar: "bg-foreground/30", title: "text-foreground" },
  success: { icon: CheckCircle2, tile: "bg-success-tint text-success", bar: "bg-success", title: "text-foreground" },
  error: { icon: XCircle, tile: "bg-destructive-tint text-destructive", bar: "bg-destructive", title: "text-foreground" },
  info: { icon: Info, tile: "bg-link-tint text-link", bar: "bg-link", title: "text-foreground" },
  warning: { icon: AlertTriangle, tile: "bg-warning-tint text-warning", bar: "bg-warning", title: "text-foreground" },
};

const LONG_DESCRIPTION = 110;

function ToastBody({ t }: { t: ToastRecord }) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const style = VARIANT_STYLE[t.variant];
  const Icon = style.icon;
  const isLong = (t.description?.length ?? 0) > LONG_DESCRIPTION;
  const copyable = t.variant === "error" || isLong;

  async function onCopy() {
    const text = t.description ? `${t.title}\n${t.description}` : t.title;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable (insecure context) */
    }
  }

  return (
    <>
      <span
        className={cn(
          "flex size-8 shrink-0 items-center justify-center rounded-xl",
          style.tile,
        )}
      >
        <Icon className="size-4" aria-hidden />
      </span>

      <div className="min-w-0 flex-1 pr-5">
        <ToastTitle className={cn("leading-5", style.title)}>{t.title}</ToastTitle>
        {t.description ? (
          <ToastDescription
            className={cn(
              "mt-0.5 break-words leading-relaxed",
              t.variant === "error" && "font-mono text-[12px]",
              isLong && !expanded && "line-clamp-2",
              isLong && expanded && "max-h-40 overflow-y-auto",
            )}
          >
            {t.description}
          </ToastDescription>
        ) : null}

        {copyable || isLong ? (
          <div className="mt-2 flex items-center gap-1.5">
            {copyable ? (
              <button
                type="button"
                onClick={() => void onCopy()}
                className="inline-flex h-7 items-center gap-1.5 rounded-full bg-muted px-2.5 text-xs font-medium text-foreground outline-none transition-colors hover:bg-surface focus-visible:ring-2 focus-visible:ring-ring"
              >
                {copied ? (
                  <Check className="size-3 text-success" aria-hidden />
                ) : (
                  <Copy className="size-3" aria-hidden />
                )}
                {copied ? "Copied" : "Copy"}
              </button>
            ) : null}
            {isLong ? (
              <button
                type="button"
                onClick={() => setExpanded((v) => !v)}
                aria-expanded={expanded}
                className="inline-flex h-7 items-center gap-1 rounded-full px-2 text-xs font-medium text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
              >
                {expanded ? "Less" : "Details"}
                <ChevronDown
                  className={cn("size-3 transition-transform", expanded && "rotate-180")}
                  aria-hidden
                />
              </button>
            ) : null}
          </div>
        ) : null}
      </div>

      <ToastClose className="right-3 top-3" />

      <span
        aria-hidden
        className={cn(
          "absolute inset-x-0 bottom-0 h-0.5 origin-left group-hover:[animation-play-state:paused]",
          style.bar,
        )}
        style={{
          animation: `toast-progress ${t.duration}ms linear forwards`,
          opacity: 0.55,
        }}
      />
    </>
  );
}

export function Toaster() {
  const toasts = useSyncExternalStore(
    subscribe,
    getToasts,
    getToastsServerSnapshot,
  );

  return (
    <ToastProvider duration={5000} label="Notifications" swipeDirection="right">
      {toasts.map((t) => (
        <ToastSurface
          key={t.id}
          open={t.open}
          duration={t.duration}
          type={t.variant === "error" ? "foreground" : "background"}
          onOpenChange={(open) => {
            if (!open) {
              dismiss(t.id);
              window.setTimeout(() => remove(t.id), 300);
            }
          }}
          className="items-start ring-1 ring-border/70 shadow-lg [animation:toast-in_200ms_ease-out] data-[state=closed]:[animation:toast-out_200ms_ease-in_forwards]"
        >
          <ToastBody t={t} />
        </ToastSurface>
      ))}
      <ToastViewport />
    </ToastProvider>
  );
}
