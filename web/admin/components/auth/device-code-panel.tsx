"use client";

import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import {
  formatCountdown,
  useCountdown,
} from "@/components/blueprint/github-health-shared";
import { Button } from "@/components/ui/button";
import { copyToClipboard } from "@/lib/api";
import { toast } from "@/lib/toast";
import { motion } from "framer-motion";
import { Check, Copy, ExternalLink } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

const EASE = [0.32, 0.72, 0, 1] as const;

type Props = {
  userCode: string;
  verificationURI: string;
  /** Epoch ms when GitHub stops accepting the code. */
  expiresAt: number;
  /** Total lifetime in seconds, for the progress bar. */
  lifetimeSeconds: number;
  onCancel: () => void;
  onExpired: () => void;
};

/**
 * Device-flow step: shows the one-time code as large tiles, links to GitHub,
 * and waits. Polling lives in the page; this only presents and counts down.
 */
export function DeviceCodePanel({
  userCode,
  verificationURI,
  expiresAt,
  lifetimeSeconds,
  onCancel,
  onExpired,
}: Props) {
  const reduce = useSafeReducedMotion();
  const [copied, setCopied] = useState(false);
  const expiresIso = useMemo(() => new Date(expiresAt).toISOString(), [expiresAt]);
  const remaining = useCountdown(expiresIso);

  const expiredRef = useRef(onExpired);
  useEffect(() => {
    expiredRef.current = onExpired;
  });
  useEffect(() => {
    if (remaining === 0) expiredRef.current();
  }, [remaining]);

  async function copy() {
    try {
      await copyToClipboard(userCode);
      setCopied(true);
      toast.success("Code copied");
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      toast.error("Could not copy the code");
    }
  }

  const chars = userCode.split("");
  const left = remaining === null ? 1 : remaining / 1000 / Math.max(1, lifetimeSeconds);

  return (
    <motion.div
      initial={reduce ? false : { opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      exit={reduce ? { opacity: 0 } : { opacity: 0, y: -8 }}
      transition={{ duration: 0.35, ease: EASE }}
      className="mt-6"
    >
      <p className="text-eyebrow text-muted-foreground">Your one-time code</p>

      <div
        className="mt-3 flex w-full items-center gap-1.5"
        role="group"
        aria-label={`Device code ${userCode.split("").join(" ")}`}
      >
        {chars.map((ch, i) =>
          ch === "-" ? (
            <span
              key={i}
              aria-hidden
              className="mx-0.5 h-0.5 w-2.5 shrink-0 rounded-full bg-border"
            />
          ) : (
            <motion.span
              key={i}
              aria-hidden
              initial={reduce ? false : { opacity: 0, rotateX: -80, y: 8 }}
              animate={{ opacity: 1, rotateX: 0, y: 0 }}
              transition={{ duration: 0.45, delay: 0.08 + i * 0.045, ease: EASE }}
              style={{ transformPerspective: 400 }}
              className="flex h-12 min-w-0 flex-1 items-center justify-center rounded-xl bg-muted font-mono text-xl text-foreground shadow-rb-1 ring-1 ring-border"
            >
              {ch}
            </motion.span>
          ),
        )}
      </div>

      <div className="mt-4 flex gap-2.5">
        <Button asChild variant="ink" size="lg" className="auth-cta flex-1">
          <a href={verificationURI} target="_blank" rel="noreferrer">
            Open GitHub
            <ExternalLink className="size-4" />
          </a>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="lg"
          onClick={() => void copy()}
          aria-label="Copy code"
        >
          {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>

      <ol className="mt-5 space-y-2 text-body-small text-muted-foreground">
        {[
          <>
            Open <span className="font-mono text-foreground">{hostOf(verificationURI)}</span>{" "}
            on any device.
          </>,
          "Enter the code above and approve Obelisk.",
          "This page continues on its own.",
        ].map((line, i) => (
          <li key={i} className="flex gap-2.5">
            <span
              aria-hidden
              className="mt-px flex size-[18px] shrink-0 items-center justify-center rounded-full bg-muted text-[10px] font-medium"
            >
              {i + 1}
            </span>
            <span>{line}</span>
          </li>
        ))}
      </ol>

      <div className="mt-5 rounded-2xl bg-muted/60 p-3.5">
        <div className="flex items-center justify-between gap-3 text-metadata">
          <span
            className="flex items-center gap-2 text-foreground"
            role="status"
            aria-live="polite"
          >
            <span aria-hidden className="flex items-center gap-1">
              <span className="auth-dot size-1.5 rounded-full bg-[color:var(--ring)]" />
              <span className="auth-dot size-1.5 rounded-full bg-[color:var(--ring)]" />
              <span className="auth-dot size-1.5 rounded-full bg-[color:var(--ring)]" />
            </span>
            Waiting for approval
          </span>
          {remaining !== null && (
            <span className="font-mono tabular-nums text-muted-foreground">
              expires in {formatCountdown(remaining)}
            </span>
          )}
        </div>
        <div className="mt-2.5 h-1 overflow-hidden rounded-full bg-border" aria-hidden>
          <motion.div
            className="h-full origin-left rounded-full bg-[color:var(--ring)]"
            initial={false}
            animate={{ scaleX: Math.min(1, Math.max(0, left)) }}
            transition={{ duration: reduce ? 0 : 0.9, ease: "linear" }}
          />
        </div>
      </div>

      <button
        type="button"
        onClick={onCancel}
        className="mt-4 cursor-pointer text-sm font-medium text-link underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Use a different sign-in method
      </button>
    </motion.div>
  );
}

function hostOf(uri: string): string {
  try {
    const u = new URL(uri);
    return `${u.host}${u.pathname === "/" ? "" : u.pathname}`;
  } catch {
    return uri;
  }
}
