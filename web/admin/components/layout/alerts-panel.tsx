"use client";

import { Button } from "@/components/ui/button";
import { FieldPassword } from "@/components/ui/field";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  rtkErrorMessage,
  useGetBlueprintStatusQuery,
  useGetUpdateStatusQuery,
  useUnlockBlueprintMutation,
} from "@/lib/api";
import { readPassphrase, writePassphrase } from "@/lib/blueprint-storage";
import { copyToClipboard } from "@/lib/copy-to-clipboard";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  ArrowRight,
  Bell,
  BellOff,
  Copy,
  Download,
  GitMerge,
  LockKeyhole,
  RefreshCw,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

type Severity = "warning" | "error";

type AlertItem = {
  id: string;
  severity: Severity;
  icon: LucideIcon;
  title: string;
  body: string;
  detail?: string;
  action?: { label: string; href: string };
  unlock?: boolean;
  update?: boolean;
};

const SEVERITY_TILE: Record<Severity, string> = {
  warning: "bg-warning-tint text-warning",
  error: "bg-destructive-tint text-destructive",
};

export function AlertsPanel() {
  const { data } = useGetBlueprintStatusQuery(undefined, { pollingInterval: 5000 });
  // Same check as `obelisk update --check`; the controller caches the GitHub lookup.
  const { data: update, refetch: recheck, isFetching: rechecking } = useGetUpdateStatusQuery(
    undefined,
    { pollingInterval: 30 * 60 * 1000 },
  );
  const [unlock, { isLoading }] = useUnlockBlueprintMutation();
  const [open, setOpen] = useState(false);
  const [passphrase, setPassphrase] = useState("");
  const tried = useRef(false);

  // Resume sync automatically when a passphrase is remembered on this device.
  useEffect(() => {
    if (!data || data.crypto_unlocked || !data.repository || tried.current) return;
    const stored = readPassphrase();
    if (!stored) return;
    tried.current = true;
    unlock({ passphrase: stored })
      .unwrap()
      .catch((e) => toast.error(rtkErrorMessage(e)));
  }, [data, unlock]);

  const alerts: AlertItem[] = [];
  if (data?.repository) {
    if (!data.crypto_unlocked) {
      alerts.push({
        id: "blueprint-locked",
        severity: "warning",
        icon: LockKeyhole,
        title: "Blueprint push is paused",
        body: "The controller restarted or was locked. Local saves still work. Enter the passphrase to resume GitHub sync.",
        detail:
          data.import_required
            ? data.remote_generation
              ? `GitHub is at generation ${data.remote_generation} (ahead of local ${data.generation}). Resolve in Platform → Blueprint after unlocking.`
              : "GitHub is ahead of this appliance. Resolve in Platform → Blueprint after unlocking."
            : data.last_push_error || undefined,
        unlock: true,
      });
    } else if (data.import_required) {
      alerts.push({
        id: "blueprint-conflict",
        severity: "error",
        icon: GitMerge,
        title: "Blueprint sync conflict",
        body: data.remote_generation
          ? `GitHub (${data.repository}) is at generation ${data.remote_generation}, ahead of this appliance (generation ${data.generation}). Choose whether to import from GitHub or overwrite GitHub with this appliance.`
          : `GitHub (${data.repository}) is ahead of this appliance (generation ${data.generation}). Choose whether to import from GitHub or overwrite GitHub with this appliance.`,
        action: { label: "Resolve conflict", href: "/platform#blueprint" },
      });
    }
  }

  if (update?.update_available) {
    alerts.push({
      id: "update-available",
      severity: "warning",
      icon: Download,
      title: "Update available",
      body: `Build ${update.latest_build} is available (installed: build ${update.installed_build}). Updating restarts the platform and must run on the appliance as root.`,
      detail: update.command,
      update: true,
    });
  }

  async function recheckUpdate() {
    try {
      const res = await recheck().unwrap();
      toast.success(
        res.update_available
          ? `Update available: build ${res.installed_build} → ${res.latest_build}`
          : res.error || "Up to date",
      );
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  async function copyUpdateCommand(command: string) {
    try {
      await copyToClipboard(command);
      toast.success("Command copied");
    } catch {
      toast.error("Could not copy the command");
    }
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    try {
      await unlock({ passphrase }).unwrap();
      writePassphrase(passphrase);
      setPassphrase("");
      toast.success("Blueprint unlocked");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  const count = alerts.length;
  const hasError = alerts.some((a) => a.severity === "error");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="icon"
          size="icon"
          className="relative"
          aria-label={count ? `Alerts, ${count} active` : "Alerts"}
          title="Alerts"
        >
          <Bell className="size-4" />
          {count > 0 ? (
            <span
              className={cn(
                "absolute -right-0.5 -top-0.5 flex size-4 items-center justify-center rounded-full text-[10px] font-semibold leading-none text-white ring-2 ring-background",
                hasError ? "bg-destructive" : "bg-warning",
              )}
            >
              {count}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={8} className="w-[24rem] max-w-[calc(100vw-2rem)] p-0">
        <div className="flex items-center justify-between px-4 py-3">
          <p className="text-section-title">Alerts</p>
          {count > 0 ? (
            <span className="text-metadata text-muted-foreground">{count} active</span>
          ) : null}
        </div>
        <div className="max-h-[70vh] overflow-y-auto border-t border-border/60">
          {count === 0 ? (
            <div className="flex flex-col items-center gap-2 px-4 py-10 text-center">
              <span className="flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
                <BellOff className="size-4" aria-hidden />
              </span>
              <p className="text-sm font-medium">All clear</p>
              <p className="text-metadata text-muted-foreground">
                Nothing needs your attention right now.
              </p>
            </div>
          ) : (
            <ul className="divide-y divide-border/60">
              {alerts.map((a) => {
                const Icon = a.icon;
                return (
                  <li key={a.id} className="flex gap-3 px-4 py-4">
                    <span
                      className={cn(
                        "flex size-8 shrink-0 items-center justify-center rounded-xl",
                        SEVERITY_TILE[a.severity],
                      )}
                    >
                      <Icon className="size-4" aria-hidden />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="text-[13px] font-semibold leading-5">{a.title}</p>
                      <p className="mt-0.5 text-metadata leading-relaxed text-muted-foreground">
                        {a.body}
                      </p>
                      {a.detail ? (
                        <p className="mt-1.5 break-words rounded-lg bg-muted px-2.5 py-1.5 text-[12px] leading-relaxed text-muted-foreground">
                          {a.detail}
                        </p>
                      ) : null}
                      {a.unlock ? (
                        <form onSubmit={(e) => void submit(e)} className="mt-3 flex items-end gap-2">
                          <FieldPassword
                            id="unlock-passphrase"
                            label="Passphrase"
                            value={passphrase}
                            onChange={(e) => setPassphrase(e.target.value)}
                            autoComplete="current-password"
                            className="min-w-0 flex-1"
                          />
                          <Button type="submit" size="sm" disabled={isLoading || passphrase.length === 0}>
                            Unlock
                          </Button>
                        </form>
                      ) : null}
                      {a.update && update ? (
                        <div className="mt-3 flex items-center gap-2">
                          <Button
                            type="button"
                            size="sm"
                            onClick={() => void copyUpdateCommand(update.command)}
                          >
                            <Copy className="size-3.5" aria-hidden />
                            Copy command
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            disabled={rechecking}
                            onClick={() => void recheckUpdate()}
                          >
                            <RefreshCw className={cn("size-3.5", rechecking && "animate-spin")} aria-hidden />
                            Check again
                          </Button>
                        </div>
                      ) : null}
                      {a.action ? (
                        <Button asChild size="sm" className="mt-3">
                          <Link href={a.action.href} onClick={() => setOpen(false)}>
                            {a.action.label}
                            <ArrowRight className="size-3.5" aria-hidden />
                          </Link>
                        </Button>
                      ) : null}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
