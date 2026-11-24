"use client";

import {
  STATE_META,
  ToneOrb,
  formatClock,
  formatCountdown,
  useCountdown,
} from "@/components/blueprint/github-health-shared";
import { Button } from "@/components/ui/button";
import {
  obeliskApi,
  useLazyGetSystemHealthQuery,
  type GitHubHealth,
} from "@/lib/api";
import { useAppDispatch } from "@/lib/store/hooks";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { ChevronDown, ExternalLink, Loader2, RefreshCw } from "lucide-react";
import { useState } from "react";

const EASE = [0.32, 0.72, 0, 1] as const;

/**
 * Floating notice for a signed-in console while GitHub is failing. Blueprint
 * sync pauses, everything else keeps working. It floats so it cannot disturb
 * the shell layout.
 */
export function GitHubStatusBanner({ health }: { health: GitHubHealth }) {
  const reduce = useReducedMotion();
  const dispatch = useAppDispatch();
  const [trigger] = useLazyGetSystemHealthQuery();
  const [open, setOpen] = useState(false);
  const [checking, setChecking] = useState(false);
  const remaining = useCountdown(health.state === "rate_limited" ? health.reset_at : undefined);
  const meta = STATE_META[health.state] ?? STATE_META.error;

  const subtitle =
    remaining !== null && health.state === "rate_limited"
      ? remaining > 0
        ? `Blueprint sync paused · resets in ${formatCountdown(remaining)}${health.reset_at ? ` (${formatClock(health.reset_at)})` : ""}`
        : "Limit reset · checking again…"
      : (health.message ?? "Blueprint sync is paused until GitHub responds.");

  async function recheck() {
    setChecking(true);
    try {
      const report = await trigger({ fresh: true }).unwrap();
      const gh = report.checks.find((c) => c.id === "github_api")?.github;
      dispatch(obeliskApi.util.invalidateTags(["Blueprint"]));
      if (gh && !gh.blocking) toast.success("GitHub is reachable again");
      else toast.info("Still unavailable", gh?.title ?? health.title);
    } catch {
      toast.info("Still unavailable", health.title);
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex justify-center px-4">
      <motion.aside
        role="status"
        aria-live="polite"
        initial={reduce ? false : { opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: reduce ? 0 : 0.35, ease: EASE }}
        className="pointer-events-auto w-full max-w-xl overflow-hidden rounded-2xl border border-border bg-card shadow-rb-2"
      >
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          className="flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
        >
          <ToneOrb state={health.state} size="sm" />
          <span className="min-w-0 flex-1">
            <span className="block truncate text-[13px] font-medium">{health.title}</span>
            <span className="block truncate text-metadata text-muted-foreground tabular-nums">
              {subtitle}
            </span>
          </span>
          <ChevronDown
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform",
              open && "rotate-180",
            )}
          />
        </button>

        <AnimatePresence initial={false}>
          {open && (
            <motion.div
              initial={reduce ? false : { height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={reduce ? { opacity: 0 } : { height: 0, opacity: 0 }}
              transition={{ duration: reduce ? 0 : 0.24, ease: EASE }}
              className="overflow-hidden"
            >
              <div className="border-t border-border px-4 pb-4 pt-3">
                {health.hint && (
                  <p className="text-body-small text-muted-foreground">{health.hint}</p>
                )}
                {health.steps && health.steps.length > 0 && (
                  <ul className="mt-3 space-y-1.5">
                    {health.steps.map((s) => (
                      <li key={s} className="flex gap-2 text-body-small">
                        <span aria-hidden className="text-muted-foreground">
                          •
                        </span>
                        <span>{s}</span>
                      </li>
                    ))}
                  </ul>
                )}
                <div className="mt-3.5 flex flex-wrap items-center gap-2">
                  {health.sso_url && (
                    <Button asChild size="sm" variant="broadcast">
                      <a href={health.sso_url} target="_blank" rel="noreferrer">
                        Authorize SSO
                        <ExternalLink className="size-3.5" />
                      </a>
                    </Button>
                  )}
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => void recheck()}
                    disabled={checking}
                  >
                    {checking ? (
                      <Loader2 className="size-3.5 animate-spin" />
                    ) : (
                      <RefreshCw className="size-3.5" />
                    )}
                    Check again
                  </Button>
                  {health.request_id && (
                    <span
                      className="ml-auto truncate font-mono text-[11px] text-muted-foreground"
                      title="Include this when contacting GitHub Support"
                    >
                      {meta.eyebrow} · {health.request_id}
                    </span>
                  )}
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.aside>
    </div>
  );
}
