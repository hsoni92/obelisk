"use client";

import { ObeliskWordmark } from "@/components/brand/obelisk-logo";
import {
  STATE_META,
  StatePill,
  ToneOrb,
  agoLabel,
  buildDiagnostics,
  checkTone,
  formatClock,
  formatCountdown,
  useCountdown,
} from "@/components/blueprint/github-health-shared";
import { Button } from "@/components/ui/button";
import { StatusDot } from "@/components/ui/status-dot";
import {
  copyToClipboard,
  obeliskApi,
  useGetSystemHealthQuery,
  type GitHubHealth,
  type SystemCheck,
} from "@/lib/api";
import { useAppDispatch } from "@/lib/store/hooks";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import {
  ChevronDown,
  Copy,
  ExternalLink,
  Loader2,
  Lock,
  RefreshCw,
} from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";

export type GitHubErrorContext = "console" | "login" | "onboard" | "callback";

const CONTEXT_COPY: Record<GitHubErrorContext, { heading: string; body: string }> = {
  console: {
    heading: "Console access is paused",
    body: "The blueprint lives on GitHub, so until it can be reached the console stays closed instead of showing an empty appliance.",
  },
  login: {
    heading: "Sign-in is unavailable",
    body: "GitHub has to approve the sign-in. It will open again automatically as soon as GitHub responds.",
  },
  onboard: {
    heading: "Onboarding is unavailable",
    body: "Creating or restoring a blueprint needs GitHub. Nothing was changed on this appliance.",
  },
  callback: {
    heading: "Sign-in could not be completed",
    body: "GitHub did not finish the sign-in. Nothing was changed on this appliance.",
  },
};

const FLOW_STEPS = ["Sign in with GitHub", "Unlock blueprint", "Restore or create"];

const EASE = [0.32, 0.72, 0, 1] as const;

type Props = {
  /** The failure to explain. A fresher live check replaces it while it is blocking. */
  health: GitHubHealth;
  context: GitHubErrorContext;
  /** Called once a live check reports GitHub is usable again. */
  onRecovered?: () => void;
  backHref?: string;
  backLabel?: string;
};

/**
 * Full-page explanation of a GitHub failure (rate limit, 401/402/403, outage,
 * no network). It replaces the sign-in and onboarding flows, which could only
 * fail the same way, and recovers on its own when GitHub does.
 */
export function GitHubErrorScreen({
  health,
  context,
  onRecovered,
  backHref,
  backLabel = "Back to sign in",
}: Props) {
  const reduce = useReducedMotion();
  const dispatch = useAppDispatch();
  const { data: report, refetch, isFetching } = useGetSystemHealthQuery(
    { fresh: true },
    { pollingInterval: 20000, refetchOnMountOrArgChange: true },
  );
  const [checking, setChecking] = useState(false);
  const [now, setNow] = useState(() => Date.now());

  const live = report?.checks.find((c) => c.id === "github_api")?.github;
  const shown: GitHubHealth = live?.blocking ? live : health;
  const recovered = Boolean(live && live.state === "ok" && !live.blocking);
  const meta = STATE_META[shown.state] ?? STATE_META.error;
  const copy = CONTEXT_COPY[context];

  const recoveredRef = useRef(onRecovered);
  useEffect(() => {
    recoveredRef.current = onRecovered;
  });
  useEffect(() => {
    if (recovered) recoveredRef.current?.();
  }, [recovered]);

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 5000);
    return () => window.clearInterval(id);
  }, []);

  // Rate-limit countdown, and the share of the wait elapsed since this screen opened.
  const remaining = useCountdown(shown.state === "rate_limited" ? shown.reset_at : undefined);
  const [openedAt] = useState(() => Date.now());
  const resetMs = shown.reset_at ? new Date(shown.reset_at).getTime() : NaN;
  const totalMs = Number.isNaN(resetMs) ? 0 : resetMs - openedAt;
  const progress =
    remaining !== null && totalMs > 0 ? Math.min(1, Math.max(0, 1 - remaining / totalMs)) : 0;

  const recheck = useCallback(async () => {
    let result: GitHubHealth | undefined;
    try {
      const res = await refetch();
      result = res.data?.checks.find((c) => c.id === "github_api")?.github;
    } catch {
      /* the failure is already on screen */
    }
    // Status polling picks the new state up and swaps this screen out.
    dispatch(obeliskApi.util.invalidateTags(["Blueprint"]));
    return result;
  }, [refetch, dispatch]);

  async function onCheckAgain() {
    setChecking(true);
    const started = Date.now();
    const result = await recheck();
    window.setTimeout(
      () => setChecking(false),
      Math.max(0, 700 - (Date.now() - started)),
    );
    if (result && !result.blocking) {
      toast.success("GitHub is reachable again");
    } else {
      toast.info("Still unavailable", result?.title ?? shown.title);
    }
  }

  // When a rate limit expires, check again without waiting for the next poll.
  const firedFor = useRef("");
  useEffect(() => {
    if (remaining !== 0 || !shown.reset_at || firedFor.current === shown.reset_at) return;
    firedFor.current = shown.reset_at;
    const id = window.setTimeout(() => void recheck(), 1500);
    return () => window.clearTimeout(id);
  }, [remaining, shown.reset_at, recheck]);

  async function onCopy() {
    try {
      await copyToClipboard(buildDiagnostics(shown, report?.checks));
      toast.success("Diagnostics copied");
    } catch {
      toast.error("Could not copy diagnostics");
    }
  }

  const rise = (delay = 0) =>
    reduce
      ? {}
      : {
          initial: { opacity: 0, y: 10 },
          animate: { opacity: 1, y: 0 },
          transition: { duration: 0.38, delay, ease: EASE },
        };

  const busy = checking || isFetching;

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10 text-foreground">
      <div className="w-full max-w-xl">
        <ObeliskWordmark />

        <motion.section
          {...rise(0)}
          aria-labelledby="gh-error-title"
          className="mt-6 overflow-hidden rounded-2xl border border-border bg-card shadow-rb-2"
        >
          <div className="p-6">
            <div className="flex items-start gap-4">
              <span className="relative mt-0.5 shrink-0">
                {!reduce && (
                  <motion.span
                    aria-hidden
                    className={cn(
                      "absolute inset-0 rounded-full",
                      meta.tone === "bad" ? "bg-destructive/30" : "bg-warning/30",
                    )}
                    animate={{ scale: [1, 1.6], opacity: [0.4, 0] }}
                    transition={{ duration: 2.4, repeat: Infinity, ease: "easeOut" }}
                  />
                )}
                <ToneOrb state={shown.state} />
              </span>
              <div className="min-w-0">
                <StatePill health={shown} />
                <h1 id="gh-error-title" role="alert" className="mt-2.5 text-page-title">
                  {shown.title}
                </h1>
                {shown.message && (
                  <p className="mt-1.5 break-words text-body text-muted-foreground">
                    {shown.message}
                  </p>
                )}
              </div>
            </div>

            {shown.hint && (
              <p className="mt-4 text-body text-muted-foreground">{shown.hint}</p>
            )}

            {remaining !== null && shown.state === "rate_limited" && (
              <div className="mt-5 rounded-xl bg-muted/60 p-4">
                <div className="flex items-end justify-between gap-4">
                  <div>
                    <p
                      className="text-display font-light tabular-nums"
                      aria-label={
                        remaining > 0
                          ? `Resets in ${formatCountdown(remaining)}`
                          : "Resetting now"
                      }
                    >
                      {remaining > 0 ? formatCountdown(remaining) : "0:00"}
                    </p>
                    <p className="mt-1 text-metadata text-muted-foreground">
                      {remaining > 0
                        ? `until the limit resets${shown.reset_at ? ` at ${formatClock(shown.reset_at)}` : ""}`
                        : "Limit reset — checking again…"}
                    </p>
                  </div>
                  {shown.limit ? (
                    <p className="text-right text-metadata text-muted-foreground">
                      <span className="block font-mono text-sm text-foreground tabular-nums">
                        {(shown.remaining ?? 0).toLocaleString()} /{" "}
                        {shown.limit.toLocaleString()}
                      </span>
                      requests left
                    </p>
                  ) : null}
                </div>
                <div
                  className="mt-3 h-1.5 overflow-hidden rounded-full bg-border"
                  role="progressbar"
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={Math.round(progress * 100)}
                  aria-label="Time until the rate limit resets"
                >
                  <motion.div
                    className="h-full rounded-full bg-warning"
                    initial={false}
                    animate={{ width: `${progress * 100}%` }}
                    transition={{ duration: reduce ? 0 : 0.9, ease: "linear" }}
                  />
                </div>
              </div>
            )}

            {shown.steps && shown.steps.length > 0 && (
              <div className="mt-5">
                <p className="text-eyebrow text-muted-foreground">What you can do</p>
                <ol className="mt-3 space-y-2.5">
                  {shown.steps.map((step, i) => (
                    <li key={step} className="flex gap-3 text-body">
                      <span
                        aria-hidden
                        className="mt-px flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-[11px] font-medium text-muted-foreground"
                      >
                        {i + 1}
                      </span>
                      <span>{step}</span>
                    </li>
                  ))}
                </ol>
              </div>
            )}

            <div className="mt-6 flex flex-wrap items-center gap-2.5">
              {shown.sso_url && (
                <Button asChild variant="broadcast">
                  <a href={shown.sso_url} target="_blank" rel="noreferrer">
                    Authorize SSO
                    <ExternalLink className="size-4" />
                  </a>
                </Button>
              )}
              <Button
                type="button"
                variant={shown.sso_url ? "outline" : "default"}
                onClick={() => void onCheckAgain()}
                disabled={busy}
              >
                {busy ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <RefreshCw className="size-4" />
                )}
                {busy ? "Checking…" : "Check again"}
              </Button>
              <Button type="button" variant="outline" onClick={() => void onCopy()}>
                <Copy className="size-4" />
                Copy diagnostics
              </Button>
              {backHref && (
                <Link href={backHref} className="ml-auto text-sm font-medium text-link">
                  {backLabel}
                </Link>
              )}
            </div>
          </div>

          <div className="flex items-center justify-between gap-3 border-t border-border bg-muted/40 px-6 py-3 text-metadata text-muted-foreground">
            <span className="flex items-center gap-2">
              <StatusDot tone="warn" pulse />
              Checking automatically · last checked {agoLabel(shown.checked_at, now) || "just now"}
            </span>
            {shown.request_id && (
              <span className="truncate font-mono" title="Include this when contacting GitHub Support">
                {shown.request_id}
              </span>
            )}
          </div>
        </motion.section>

        <motion.section
          {...rise(0.08)}
          aria-label={copy.heading}
          className="mt-4 rounded-2xl border border-border bg-card p-5 shadow-rb-1"
        >
          <div className="flex items-center gap-2">
            <Lock className="size-4 text-muted-foreground" strokeWidth={1.75} />
            <h2 className="text-section-title">{copy.heading}</h2>
          </div>
          <p className="mt-1.5 text-body-small text-muted-foreground">{copy.body}</p>
          <ul className="mt-4 grid gap-2 sm:grid-cols-3" aria-label="Unavailable steps">
            {FLOW_STEPS.map((label) => (
              <li
                key={label}
                className="flex items-center gap-2 rounded-xl border border-dashed border-border px-3 py-2.5 text-[13px] text-muted-foreground"
              >
                <Lock className="size-3.5 shrink-0" strokeWidth={1.75} />
                <span className="truncate">{label}</span>
              </li>
            ))}
          </ul>
        </motion.section>

        <motion.div {...rise(0.16)}>
          <ServiceChecks
            checks={report?.checks}
            loading={!report}
            fetching={isFetching}
            onRerun={() => void onCheckAgain()}
          />
        </motion.div>
      </div>
    </div>
  );
}

function ServiceChecks({
  checks,
  loading,
  fetching,
  onRerun,
}: {
  checks?: SystemCheck[];
  loading: boolean;
  fetching: boolean;
  onRerun: () => void;
}) {
  const reduce = useReducedMotion();
  const [open, setOpen] = useState(true);
  const failing = checks?.filter((c) => c.state === "fail").length ?? 0;
  const warning = checks?.filter((c) => c.state === "warn").length ?? 0;
  const summary = loading
    ? "Checking…"
    : failing + warning === 0
      ? "All clear"
      : [failing ? `${failing} failing` : "", warning ? `${warning} warning` : ""]
          .filter(Boolean)
          .join(" · ");

  return (
    <section
      aria-label="Service checks"
      className="mt-4 overflow-hidden rounded-2xl border border-border bg-card shadow-rb-1"
    >
      <div className="flex items-center justify-between gap-3 px-5 py-3.5">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <h2 className="text-section-title">Service checks</h2>
          <span className="truncate text-metadata text-muted-foreground">{summary}</span>
          <ChevronDown
            className={cn(
              "ml-auto size-4 shrink-0 text-muted-foreground transition-transform",
              open && "rotate-180",
            )}
          />
        </button>
        <Button
          type="button"
          variant="icon"
          size="sm"
          onClick={onRerun}
          disabled={fetching}
          aria-label="Run service checks again"
        >
          <RefreshCw className={cn(fetching && "animate-spin")} />
        </Button>
      </div>

      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={reduce ? false : { height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={reduce ? { opacity: 0 } : { height: 0, opacity: 0 }}
            transition={{ duration: reduce ? 0 : 0.26, ease: EASE }}
            className="overflow-hidden"
          >
            <ul className="divide-y divide-border border-t border-border">
              {loading && (
                <li className="px-5 py-4 text-body-small text-muted-foreground">
                  Running checks…
                </li>
              )}
              {checks?.map((c) => (
                <li key={c.id} className="px-5 py-3">
                  <div className="flex items-start gap-3">
                    <StatusDot tone={checkTone(c.state)} className="mt-1.5" />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-baseline justify-between gap-x-3">
                        <p className="text-[13px] font-medium">{c.name}</p>
                        {c.latency_ms ? (
                          <p className="font-mono text-[11px] text-muted-foreground tabular-nums">
                            {c.latency_ms} ms
                          </p>
                        ) : null}
                      </div>
                      <p className="text-body-small text-muted-foreground">{c.summary}</p>
                      {/* The GitHub API failure is already explained in full above. */}
                      {(c.state === "fail" || c.state === "warn") &&
                        c.id !== "github_api" &&
                        c.detail && (
                          <p className="mt-1 break-words text-body-small text-muted-foreground">
                            {c.detail}
                          </p>
                        )}
                      {(c.state === "fail" || c.state === "warn") &&
                        c.id !== "github_api" &&
                        c.hint && (
                          <p className="mt-1 text-body-small text-foreground">→ {c.hint}</p>
                        )}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </motion.div>
        )}
      </AnimatePresence>
    </section>
  );
}
