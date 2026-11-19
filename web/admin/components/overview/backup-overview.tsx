"use client";

import { BackupOverviewSkeleton } from "@/components/skeletons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeading } from "@/components/ui/card";
import type { BackupSummary, BackupTone } from "@/lib/backup-summary";
import { formatBytes } from "@/lib/format-bytes";
import { relativeTime } from "@/lib/overview-series";
import type { BackupConfigResponse, BackupDestinationView } from "@/lib/types";
import { cn } from "@/lib/utils";
import { AlertCircle, DatabaseBackup, HardDrive } from "lucide-react";

const BAR: Record<BackupTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  live: "bg-broadcast",
  default: "bg-muted-foreground/50",
};

function destTone(d: BackupDestinationView): BackupTone {
  if (!d.enabled) return "default";
  if (d.ready) return "success";
  return d.status === "offline" ? "live" : "warning";
}

export function BackupOverview({
  config,
  summary,
  onOpenDR,
}: {
  config?: BackupConfigResponse;
  summary: BackupSummary | null;
  onOpenDR: () => void;
}) {
  if (!config) return <BackupOverviewSkeleton />;

  const dests = config?.destinations ?? [];
  const p = config?.policy;

  return (
    <Card className="overflow-hidden">
      <CardHeading
        icon={DatabaseBackup}
        tone="blue"
        title="Backups"
        description={summary?.detail}
        actions={
          <>
            {summary && <Badge variant={summary.tone}>{summary.label}</Badge>}
            <Button size="sm" variant="outline" onClick={onOpenDR}>
              Manage
            </Button>
          </>
        }
      />

      {p && summary && (
        <dl className="grid grid-cols-2 gap-px border-t border-border bg-border text-[13px] md:grid-cols-4">
          <Fact
            label="Total size"
            value={summary.totalBytes ? formatBytes(summary.totalBytes) : "0 B"}
            sub={`${summary.totalObjects.toLocaleString()} objects`}
          />
          <Fact
            label="Last backup"
            value={
              summary.lastBackupAt ? relativeTime(summary.lastBackupAt) : "never"
            }
            sub={
              summary.lastBackupAt
                ? new Date(summary.lastBackupAt).toLocaleString()
                : "no run recorded"
            }
          />
          <Fact
            label="Schedule"
            value={p.schedule || "—"}
            sub={p.include_config ? "incl. config" : "data only"}
            mono
          />
          <Fact
            label="Retention"
            value={`${p.data_retention_days} days`}
            sub={`${p.replication_factor}× replication`}
          />
        </dl>
      )}

      <div className="border-t border-border">
        {dests.length === 0 && summary && (
          <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
            <HardDrive className="size-5 text-muted-foreground" aria-hidden />
            <p className="text-sm text-muted-foreground">
              No off-site destinations yet. Add S3 or R2 to start replicating.
            </p>
            <Button size="sm" onClick={onOpenDR}>
              Add destination
            </Button>
          </div>
        )}
        <ul className="divide-y divide-border">
          {dests.map((d) => {
            const tone = destTone(d);
            const share = summary?.maxBytes
              ? Math.max(2, ((d.bytes ?? 0) / summary.maxBytes) * 100)
              : 0;
            return (
              <li key={d.id} className="px-4 py-3">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <p className="font-semibold">{d.label}</p>
                  <Badge variant={tone}>{d.enabled ? d.status : "disabled"}</Badge>
                  <span className="text-metadata text-muted-foreground">
                    {d.provider ? `${d.provider} · ` : ""}
                    {d.bucket}
                  </span>
                  <span className="ml-auto tabular-nums text-sm font-medium">
                    {formatBytes(d.bytes)}
                    {d.objects ? (
                      <span className="ml-1.5 text-metadata font-normal text-muted-foreground">
                        {d.objects.toLocaleString()} obj
                      </span>
                    ) : null}
                  </span>
                </div>
                <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
                  <div
                    className={cn("h-full rounded-full", BAR[tone])}
                    style={{ width: `${share}%` }}
                  />
                </div>
                <p className="mt-1.5 text-metadata text-muted-foreground">
                  Last upload{" "}
                  {d.last_upload_at ? relativeTime(d.last_upload_at) : "never"}
                  {d.schedule_lag ? ` · lag ${d.schedule_lag}` : ""}
                  {d.last_test_at
                    ? ` · tested ${relativeTime(d.last_test_at)} ${d.last_test_ok ? "✓" : "✗"}`
                    : ""}
                </p>
                {(d.last_error || (d.last_test_ok === false && d.last_test_message)) && (
                  <p className="mt-1 flex items-start gap-1.5 text-metadata text-destructive">
                    <AlertCircle className="mt-px size-3.5 shrink-0" aria-hidden />
                    <span className="break-words">
                      {d.last_error || d.last_test_message}
                    </span>
                  </p>
                )}
              </li>
            );
          })}
        </ul>
      </div>
    </Card>
  );
}

function Fact({
  label,
  value,
  sub,
  mono,
}: {
  label: string;
  value: string;
  sub?: string;
  mono?: boolean;
}) {
  return (
    <div className="bg-card px-4 py-3">
      <dt className="text-eyebrow text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "mt-0.5 truncate text-base font-semibold tabular-nums",
          mono && "font-mono text-sm",
        )}
        title={value}
      >
        {value}
      </dd>
      {sub && (
        <dd className="truncate text-metadata text-muted-foreground">{sub}</dd>
      )}
    </div>
  );
}
