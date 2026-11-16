"use client";

import {
  rtkErrorMessage,
  statusPollOptions,
  useGetBlueprintStatusQuery,
  useGetStatusQuery,
  useLogoutGitHubMutation,
} from "@/lib/api";
import { forgetPassphrase } from "@/lib/blueprint-storage";
import { relativeTime } from "@/lib/overview-series";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { AnimatePresence, motion } from "framer-motion";
import { Loader2, LogOut, Server } from "lucide-react";
import { useRouter } from "next/navigation";
import * as React from "react";

type Tone = "ok" | "warn" | "bad" | "idle";

const DOT: Record<Tone, string> = {
  ok: "bg-broadcast shadow-[0_0_0_3px_rgb(215_242_73/0.22)]",
  warn: "bg-[#f5c27a] shadow-[0_0_0_3px_rgb(245_194_122/0.22)]",
  bad: "bg-[#ff9a90] shadow-[0_0_0_3px_rgb(255_154_144/0.22)]",
  idle: "bg-white/40",
};

function useSidebarStatus() {
  const { data } = useGetBlueprintStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const { data: status } = useGetStatusQuery(undefined, statusPollOptions);

  let tone: Tone = "idle";
  let label = "Not connected";
  if (data?.repository) {
    const s = data.crypto_unlocked ? data.sync_status : "push_paused";
    if (s === "synced") {
      tone = "ok";
      label = `Synced · g${data.generation}`;
    } else if (s === "failed") {
      tone = "bad";
      label = "Sync failed";
    } else if (s === "push_paused") {
      tone = "warn";
      label = "Push paused";
    } else {
      tone = "warn";
      label = s.replaceAll("_", " ");
      label = label.charAt(0).toUpperCase() + label.slice(1);
    }
  }
  const profile = status?.profile
    ? status.profile.charAt(0).toUpperCase() + status.profile.slice(1)
    : "—";
  return { data, tone, label, profile };
}

function useSignOut() {
  const router = useRouter();
  const [logout] = useLogoutGitHubMutation();
  const [busy, setBusy] = React.useState(false);
  const signOut = React.useCallback(async () => {
    setBusy(true);
    forgetPassphrase();
    try {
      await logout().unwrap();
    } catch (e) {
      toast.error("Sign out failed", rtkErrorMessage(e));
    }
    router.replace("/login");
  }, [logout, router]);
  return { signOut, busy };
}

function Avatar({ login, className }: { login?: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "flex shrink-0 items-center justify-center rounded-full bg-broadcast font-medium uppercase text-broadcast-foreground",
        className,
      )}
    >
      {login?.[0] ?? "?"}
    </span>
  );
}

export function SidebarFooter({ rail }: { rail: boolean }) {
  const { data, tone, label, profile } = useSidebarStatus();
  const { signOut, busy } = useSignOut();

  return (
    <div className="mt-auto">
      <AnimatePresence initial={false} mode="wait">
        {!rail ? (
          <motion.div
            key="card"
            initial={{ opacity: 0, y: 8 }}
            animate={{
              opacity: 1,
              y: 0,
              transition: { delay: 0.18, duration: 0.2 },
            }}
            exit={{ opacity: 0, y: 8, transition: { duration: 0.1 } }}
            className="w-[220px] overflow-clip rounded-[28px] bg-feature p-4 text-feature-foreground shadow-rb-2"
          >
            <div className="flex items-center gap-3">
              <Avatar login={data?.login} className="size-10 text-base" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium" title={data?.login}>
                  {data?.login ? `@${data.login}` : "Not signed in"}
                </p>
                <p className="truncate text-xs text-white/65">GitHub account</p>
              </div>
              <button
                type="button"
                onClick={() => void signOut()}
                disabled={busy}
                aria-label="Sign out"
                title="Sign out"
                className="flex size-8 shrink-0 items-center justify-center rounded-full bg-white/15 text-white transition-colors hover:bg-white/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/70 disabled:opacity-60"
              >
                {busy ? (
                  <Loader2 className="size-3.5 animate-spin" aria-hidden />
                ) : (
                  <LogOut className="size-3.5" aria-hidden />
                )}
              </button>
            </div>

            <dl className="mt-4 space-y-2.5 rounded-2xl bg-white/10 px-3.5 py-3 text-xs">
              <div>
                <dt className="text-eyebrow text-[10px] text-white/55">
                  Blueprint
                </dt>
                <dd className="mt-1 flex items-center gap-2">
                  <span
                    className={cn("size-2 shrink-0 rounded-full", DOT[tone])}
                    aria-hidden
                  />
                  <span className="truncate text-sm font-medium" role="status">
                    {label}
                  </span>
                </dd>
                <dd className="mt-0.5 truncate pl-4 text-white/65">
                  {data?.last_sync_at
                    ? `Pushed ${relativeTime(data.last_sync_at)}`
                    : data?.repository
                      ? "No push yet"
                      : "Connect GitHub to sync"}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-2 border-t border-white/10 pt-2.5">
                <dt className="flex items-center gap-1.5 text-white/60">
                  <Server className="size-3" aria-hidden />
                  Deployment
                </dt>
                <dd className="font-medium">{profile}</dd>
              </div>
            </dl>
          </motion.div>
        ) : (
          <motion.div
            key="rail"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1, transition: { delay: 0.2, duration: 0.2 } }}
            exit={{ opacity: 0, transition: { duration: 0.1 } }}
            className="ml-2 flex w-10 flex-col items-center gap-1.5 rounded-full bg-feature p-1 shadow-rb-2"
          >
            <span
              className="relative"
              title={`${data?.login ? `@${data.login} · ` : ""}${label} · ${profile}`}
            >
              <Avatar login={data?.login} className="size-8 text-xs" />
              <span
                className={cn(
                  "absolute -bottom-0.5 -right-0.5 size-2.5 rounded-full ring-2 ring-feature",
                  DOT[tone],
                )}
              />
            </span>
            <button
              type="button"
              onClick={() => void signOut()}
              disabled={busy}
              aria-label="Sign out"
              title="Sign out"
              className="flex size-8 items-center justify-center rounded-full text-white/85 transition-colors hover:bg-white/15 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/70 disabled:opacity-60"
            >
              <LogOut className="size-4" aria-hidden />
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
