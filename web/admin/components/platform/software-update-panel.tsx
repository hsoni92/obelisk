"use client";

import { Button } from "@/components/ui/button";
import { rtkErrorMessage, useApplyUpdateMutation, useGetUpdateStatusQuery } from "@/lib/api";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { CheckCircle2, Download, Loader2, RefreshCw, TriangleAlert } from "lucide-react";
import { useState } from "react";

function Build({ label, value, emphasis }: { label: string; value?: number; emphasis?: boolean }) {
  return (
    <div className="min-w-0 flex-1 rounded-lg bg-muted/60 px-3 py-2.5">
      <p className="text-metadata text-muted-foreground">{label}</p>
      <p className={cn("mt-0.5 text-lg font-semibold tabular-nums", emphasis && "text-broadcast")}>
        {value ? `Build ${value}` : "—"}
      </p>
    </div>
  );
}

export function SoftwareUpdatePanel() {
  const [updating, setUpdating] = useState(false);
  const {
    data: update,
    refetch,
    isFetching,
    isLoading,
  } = useGetUpdateStatusQuery(undefined, { pollingInterval: updating ? 4000 : 5 * 60 * 1000 });
  const [applyUpdate, { isLoading: starting }] = useApplyUpdateMutation();

  const phase = update?.apply?.phase;
  // `updating` covers the gap after clicking, before the first poll reports the Job.
  const applying = phase === "running" || (updating && phase !== "failed" && phase !== "succeeded");
  const failed = phase === "failed" && !applying;
  if (applying !== updating && phase) setUpdating(applying);

  async function check() {
    try {
      const res = await refetch().unwrap();
      toast.success(
        res.update_available
          ? `Update available: build ${res.latest_build}`
          : res.error || "You are up to date",
      );
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  async function install() {
    try {
      await applyUpdate().unwrap();
      setUpdating(true);
      toast.success("Update started");
      void refetch();
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    }
  }

  let tone = "bg-muted text-muted-foreground";
  let Icon = CheckCircle2;
  let headline = "You're up to date";
  let sub = "This appliance is running the latest published build.";
  if (isLoading) {
    Icon = Loader2;
    headline = "Checking for updates…";
    sub = "";
  } else if (applying) {
    tone = "bg-warning-tint text-warning";
    Icon = Loader2;
    headline = `Updating to build ${update?.apply?.target_build ?? update?.latest_build ?? ""}…`;
    sub = "The platform is being updated and will restart. This page reconnects by itself.";
  } else if (failed) {
    tone = "bg-destructive-tint text-destructive";
    Icon = TriangleAlert;
    headline = "The last update failed";
    sub = "Nothing was lost. You can retry, or run it on the appliance.";
  } else if (update?.update_available) {
    tone = "bg-warning-tint text-warning";
    Icon = Download;
    headline = `Build ${update.latest_build} is available`;
    sub = "Updating restarts the platform for a short while. Your data is kept.";
  } else if (update && !update.checkable) {
    Icon = TriangleAlert;
    headline = "Updates can't be checked";
    sub = update.error || "This install is not managed by releases (source build).";
  } else if (update?.error) {
    Icon = TriangleAlert;
    headline = "Couldn't reach the release server";
    sub = update.error;
  } else {
    tone = "bg-success-tint text-success";
  }

  const canInstall = !!update?.update_available || failed;

  return (
    <section aria-labelledby="software-update-heading">
      <h2 id="software-update-heading" className="text-page-title">
        Software update
      </h2>
      <p className="mt-0.5 text-body text-muted-foreground">
        Keep this appliance on the latest Obelisk release.
      </p>

      <div className="mt-4 rounded-xl border border-border bg-card p-4 shadow-rb-2">
        <div className="flex items-start gap-3">
          <span className={cn("flex size-10 shrink-0 items-center justify-center rounded-xl", tone)}>
            <Icon className={cn("size-5", (isLoading || applying) && "animate-spin")} aria-hidden />
          </span>
          <div className="min-w-0">
            <p className="text-section-title">{headline}</p>
            {sub ? <p className="mt-0.5 text-metadata text-muted-foreground">{sub}</p> : null}
          </div>
        </div>

        {update?.checkable ? (
          <div className="mt-4 flex gap-3">
            <Build label="Installed" value={update.installed_build} />
            <Build label="Latest" value={update.latest_build} emphasis={update.update_available} />
          </div>
        ) : null}

        {failed && update?.apply?.log_tail ? (
          <pre className="mt-4 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-muted px-3 py-2 text-[12px] leading-relaxed text-muted-foreground">
            {update.apply.log_tail}
          </pre>
        ) : null}

        <div className="mt-4 flex flex-wrap items-center gap-2">
          {canInstall || applying ? (
            <Button type="button" disabled={starting || applying} onClick={() => void install()}>
              {starting || applying ? (
                <Loader2 className="size-4 animate-spin" aria-hidden />
              ) : (
                <Download className="size-4" aria-hidden />
              )}
              {applying ? "Updating…" : failed ? "Retry update" : "Install update"}
            </Button>
          ) : null}
          <Button
            type="button"
            variant="outline"
            disabled={isFetching || applying}
            onClick={() => void check()}
          >
            <RefreshCw className={cn("size-4", isFetching && "animate-spin")} aria-hidden />
            Check for updates
          </Button>
        </div>
      </div>

      {update ? (
        <dl className="mt-4 grid grid-cols-[auto_minmax(0,1fr)] gap-x-6 gap-y-1.5 text-metadata">
          <dt className="text-muted-foreground">Install type</dt>
          <dd>{update.source === "dev" ? "Source build (developer)" : "Release"}</dd>
          <dt className="text-muted-foreground">Last checked</dt>
          <dd>{new Date(update.checked_at).toLocaleString()}</dd>
          <dt className="text-muted-foreground">Manual command</dt>
          <dd className="break-all font-mono">{update.command}</dd>
        </dl>
      ) : null}
    </section>
  );
}
