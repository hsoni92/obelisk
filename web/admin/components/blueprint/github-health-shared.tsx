"use client";

import type {
  GitHubHealth,
  GitHubHealthState,
  SystemCheck,
  SystemCheckState,
} from "@/lib/api";
import type { StatusTone } from "@/components/ui/status-dot";
import { cn } from "@/lib/utils";
import {
  CircleAlert,
  CircleCheck,
  CreditCard,
  Gauge,
  KeyRound,
  ServerCrash,
  ShieldAlert,
  WifiOff,
  type LucideIcon,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

type Tone = "warn" | "bad" | "ok";

type StateMeta = {
  icon: LucideIcon;
  tone: Tone;
  /** Short label for the pill above the title. */
  eyebrow: string;
};

export const STATE_META: Record<GitHubHealthState, StateMeta> = {
  rate_limited: { icon: Gauge, tone: "warn", eyebrow: "Rate limit" },
  forbidden: { icon: ShieldAlert, tone: "bad", eyebrow: "Access denied" },
  unauthorized: { icon: KeyRound, tone: "bad", eyebrow: "Sign-in rejected" },
  payment_required: { icon: CreditCard, tone: "bad", eyebrow: "Billing" },
  unavailable: { icon: ServerCrash, tone: "warn", eyebrow: "GitHub outage" },
  unreachable: { icon: WifiOff, tone: "warn", eyebrow: "No connection" },
  error: { icon: CircleAlert, tone: "bad", eyebrow: "Error" },
  ok: { icon: CircleCheck, tone: "ok", eyebrow: "Healthy" },
};

/** Compact label for the sidebar and badges. */
export const GITHUB_SHORT_LABEL: Record<GitHubHealthState, string> = {
  rate_limited: "GitHub rate limited",
  forbidden: "GitHub access denied",
  unauthorized: "GitHub sign-in rejected",
  payment_required: "GitHub billing issue",
  unavailable: "GitHub outage",
  unreachable: "GitHub unreachable",
  error: "GitHub error",
  ok: "GitHub OK",
};

const TONE_SURFACE: Record<Tone, string> = {
  warn: "bg-warning-tint text-warning",
  bad: "bg-destructive-tint text-destructive",
  ok: "bg-success-tint text-success",
};

const TONE_RING: Record<Tone, string> = {
  warn: "ring-warning/25",
  bad: "ring-destructive/25",
  ok: "ring-success/25",
};

/** Tinted circular icon used by the error screen and the banner. */
export function ToneOrb({
  state,
  size = "lg",
  className,
}: {
  state: GitHubHealthState;
  size?: "sm" | "lg";
  className?: string;
}) {
  const meta = STATE_META[state] ?? STATE_META.error;
  const Icon = meta.icon;
  return (
    <span
      aria-hidden
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full ring-4",
        TONE_SURFACE[meta.tone],
        TONE_RING[meta.tone],
        size === "lg" ? "size-12" : "size-8 ring-2",
        className,
      )}
    >
      <Icon className={size === "lg" ? "size-5" : "size-4"} strokeWidth={1.75} />
    </span>
  );
}

/** Small pill: "Rate limit · HTTP 403". */
export function StatePill({ health }: { health: GitHubHealth }) {
  const meta = STATE_META[health.state] ?? STATE_META.error;
  return (
    <span
      className={cn(
        "text-eyebrow inline-flex items-center gap-1.5 rounded-full px-2.5 py-1",
        TONE_SURFACE[meta.tone],
      )}
    >
      GitHub · {meta.eyebrow}
      {health.http_status ? (
        <span className="font-mono normal-case tracking-normal opacity-80">
          {health.http_status}
        </span>
      ) : null}
    </span>
  );
}

export function checkTone(state: SystemCheckState): StatusTone {
  switch (state) {
    case "ok":
      return "ok";
    case "warn":
      return "warn";
    case "fail":
      return "bad";
    default:
      return "idle";
  }
}

/**
 * Milliseconds until `resetAt`, re-evaluated every second, or null when there
 * is no reset time. Reaches 0 and stays there.
 */
export function useCountdown(resetAt?: string): number | null {
  const target = useMemo(() => {
    if (!resetAt) return null;
    const t = new Date(resetAt).getTime();
    return Number.isNaN(t) ? null : t;
  }, [resetAt]);
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (target === null) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [target]);
  return target === null ? null : Math.max(0, target - now);
}

/** 754000 → "12:34", 4_000_000 → "1:06:40". */
export function formatCountdown(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}

/** Local wall-clock time for an ISO timestamp, for example "10:57 PM" or "22:57". */
export function formatClock(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}

/** "just now", "12s ago", "3m ago". */
export function agoLabel(iso: string | undefined, now: number): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const s = Math.max(0, Math.round((now - t) / 1000));
  if (s < 5) return "just now";
  if (s < 60) return `${s}s ago`;
  return `${Math.round(s / 60)}m ago`;
}

const MARK: Record<SystemCheckState, string> = {
  ok: "[ok]  ",
  warn: "[warn]",
  fail: "[fail]",
  off: "[off] ",
};

/** Plain-text report to paste into a ticket. Contains no secrets. */
export function buildDiagnostics(
  health: GitHubHealth,
  checks: SystemCheck[] | undefined,
): string {
  const lines = [
    "Obelisk GitHub diagnostics",
    `State:      ${health.state} (${health.blocking ? "blocking" : "non-blocking"})`,
  ];
  if (health.http_status) lines.push(`HTTP:       ${health.http_status}`);
  lines.push(`Title:      ${health.title}`);
  if (health.message) lines.push(`Message:    ${health.message}`);
  if (health.reset_at) lines.push(`Resets at:  ${health.reset_at}`);
  if (health.limit) {
    lines.push(`Quota:      ${health.remaining ?? "?"}/${health.limit}`);
  }
  if (health.request_id) lines.push(`Request id: ${health.request_id}`);
  lines.push(`Checked:    ${health.checked_at}`);
  if (checks?.length) {
    lines.push("", "Checks:");
    for (const c of checks) {
      lines.push(`  ${MARK[c.state]} ${c.name}: ${c.summary}`);
      if (c.detail) lines.push(`           ${c.detail}`);
    }
  }
  return lines.join("\n");
}
