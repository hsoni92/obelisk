"use client";

import { BackupOverviewSkeleton } from "@/components/skeletons";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeading } from "@/components/ui/card";
import { CopyableId } from "@/components/ui/copyable-id";
import type { BackupSummary, BackupTone } from "@/lib/backup-summary";
import { copyToClipboard } from "@/lib/copy-to-clipboard";
import { formatBytes } from "@/lib/format-bytes";
import { relativeTime } from "@/lib/overview-series";
import type { BackupConfigResponse, BackupDestinationView } from "@/lib/types";
import { cn } from "@/lib/utils";
import {
  Check,
  ChevronDown,
  CircleCheck,
  CircleX,
  Cloud,
  CloudOff,
  Copy,
  DatabaseBackup,
  HardDrive,
  History,
  Layers,
  Repeat,
  Timer,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";

const BAR: Record<BackupTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  live: "bg-destructive",
  default: "bg-muted-foreground/50",
};
const TILE: Record<BackupTone, string> = {
  success: "bg-success-tint text-success",
  warning: "bg-warning-tint text-warning",
  live: "bg-destructive-tint text-destructive",
  default: "bg-muted text-muted-foreground",
};

function destTone(d: BackupDestinationView): BackupTone {
  if (!d.enabled) return "default";
  if (d.ready) return "success";
  return d.status === "offline" ? "live" : "warning";
}

/** Plain-English cron for the common shapes; raw expression stays as the subtitle. */
function humanCron(expr?: string) {
  if (!expr) return "—";
  const [min, hour, dom, mon, dow] = expr.trim().split(/\s+/);
  if (dom === "*" && mon === "*" && dow === "*") {
    const every = hour?.match(/^\*\/(\d+)$/);
    if (min === "0" && every) return `Every ${every[1]} hours`;
    if (hour === "*" && min?.match(/^\*\/(\d+)$/)) return `Every ${min.slice(2)} minutes`;
    if (/^\d+$/.test(min ?? "") && /^\d+$/.test(hour ?? ""))
      return `Daily at ${hour.padStart(2, "0")}:${min.padStart(2, "0")}`;
  }
  return expr;
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

  const sizeShare = (d: BackupDestinationView) =>
    summary?.totalBytes ? ((d.bytes ?? 0) / summary.totalBytes) * 100 : 0;

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
        <div className="space-y-5 px-6 py-6">
          {/* Hero: size + last backup, with a per-destination split */}
          <div className="grid gap-6 sm:grid-cols-[1fr_auto] sm:items-end">
            <div>
              <p className="text-eyebrow text-muted-foreground">Stored off-site</p>
              <p className="mt-1 flex items-baseline gap-2">
                <span className="text-4xl font-light leading-none tabular-nums">
                  {summary.totalBytes ? formatBytes(summary.totalBytes) : "0 B"}
                </span>
                <span className="text-sm text-muted-foreground">
                  {summary.totalObjects.toLocaleString()} objects
                </span>
              </p>
            </div>
            <div className="sm:text-right">
              <p className="text-eyebrow text-muted-foreground">Last backup</p>
              <p className="mt-1 text-lg font-medium leading-none">
                {summary.lastBackupAt ? relativeTime(summary.lastBackupAt) : "never"}
              </p>
              <p className="mt-1 text-metadata text-muted-foreground">
                {summary.lastBackupAt
                  ? new Date(summary.lastBackupAt).toLocaleString()
                  : "no run recorded"}
              </p>
            </div>
          </div>

          {summary.totalBytes > 0 && dests.length > 1 ? (
            <div
              className="flex h-2 gap-0.5 overflow-hidden rounded-full"
              role="img"
              aria-label="Storage split across destinations"
            >
              {dests
                .filter((d) => (d.bytes ?? 0) > 0)
                .map((d) => (
                  <span
                    key={d.id}
                    title={`${d.label}: ${formatBytes(d.bytes)}`}
                    className={cn("h-full rounded-full", BAR[destTone(d)])}
                    style={{ width: `${sizeShare(d)}%` }}
                  />
                ))}
            </div>
          ) : null}

          {/* Policy */}
          <dl className="grid grid-cols-2 gap-x-6 gap-y-4 border-t border-border/60 pt-5 md:grid-cols-4">
            <Policy icon={Repeat} label="Schedule" value={humanCron(p.schedule)} sub={p.schedule} mono />
            <Policy icon={Timer} label="Retention" value={`${p.data_retention_days} days`} sub="per snapshot" />
            <Policy icon={Layers} label="Replication" value={`${p.replication_factor}×`} sub={`${summary.enabled.length} destination${summary.enabled.length === 1 ? "" : "s"}`} />
            <Policy icon={History} label="Scope" value={p.include_config ? "Data + config" : "Data only"} sub="what is captured" />
          </dl>
        </div>
      )}

      <div className="border-t border-border/60">
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
        {dests.length > 0 ? (
          <p className="px-6 pb-1 pt-4 text-eyebrow text-muted-foreground">
            Destinations
          </p>
        ) : null}
        <ul className="divide-y divide-border/60">
          {dests.map((d) => (
            <DestinationRow key={d.id} d={d} />
          ))}
        </ul>
      </div>
    </Card>
  );
}

function Policy({
  icon: Icon,
  label,
  value,
  sub,
  mono,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  sub?: string;
  mono?: boolean;
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
        <Icon className="size-4" aria-hidden />
      </span>
      <div className="min-w-0">
        <dt className="text-eyebrow text-muted-foreground">{label}</dt>
        <dd className="mt-0.5 truncate text-sm font-medium" title={value}>
          {value}
        </dd>
        {sub ? (
          <dd className={cn("truncate text-[11px] text-muted-foreground", mono && "font-mono")}>
            {sub}
          </dd>
        ) : null}
      </div>
    </div>
  );
}

function Metric({
  label,
  children,
  tone,
}: {
  label: string;
  children: React.ReactNode;
  tone?: "ok" | "bad";
}) {
  return (
    <div className="min-w-0">
      <dt className="text-[11px] text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "mt-0.5 flex items-center gap-1 truncate text-[13px] font-medium tabular-nums",
          tone === "ok" && "text-success",
          tone === "bad" && "text-destructive",
        )}
      >
        {children}
      </dd>
    </div>
  );
}

function DestinationRow({ d }: { d: BackupDestinationView }) {
  const tone = destTone(d);
  const Icon = tone === "live" ? CloudOff : Cloud;
  const error = d.last_error || (d.last_test_ok === false ? d.last_test_message : "");
  const overdue = !!d.schedule_lag && !/on schedule/i.test(d.schedule_lag);

  return (
    <li className="px-6 py-5">
      <div className="flex items-start gap-3">
        <span
          className={cn(
            "flex size-10 shrink-0 items-center justify-center rounded-xl",
            TILE[tone],
          )}
        >
          <Icon className="size-5" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold capitalize">{d.label}</p>
            <Badge variant={tone}>{d.enabled ? d.status : "disabled"}</Badge>
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-metadata text-muted-foreground">
            {d.provider ? <span className="uppercase tracking-wide">{d.provider}</span> : null}
            {d.provider ? <span aria-hidden>·</span> : null}
            <CopyableId value={d.bucket} label="bucket name" />
          </div>
        </div>
        <div className="shrink-0 text-right">
          <p className="text-lg font-light leading-none tabular-nums">{formatBytes(d.bytes)}</p>
          <p className="mt-1 text-[11px] text-muted-foreground">
            {(d.objects ?? 0).toLocaleString()} objects
          </p>
        </div>
      </div>

      <dl className="mt-4 grid grid-cols-3 gap-4 pl-[3.25rem]">
        <Metric label="Last upload">
          {d.last_upload_at ? relativeTime(d.last_upload_at) : "never"}
        </Metric>
        <Metric label="Schedule" tone={d.schedule_lag ? (overdue ? "bad" : "ok") : undefined}>
          {d.schedule_lag ? (
            <>
              {overdue ? (
                <TriangleAlert className="size-3.5 shrink-0" aria-hidden />
              ) : (
                <CircleCheck className="size-3.5 shrink-0" aria-hidden />
              )}
              <span className="truncate">{overdue ? d.schedule_lag : "On schedule"}</span>
            </>
          ) : (
            "—"
          )}
        </Metric>
        <Metric
          label="Connection test"
          tone={d.last_test_at ? (d.last_test_ok ? "ok" : "bad") : undefined}
        >
          {d.last_test_at ? (
            <>
              {d.last_test_ok ? (
                <CircleCheck className="size-3.5 shrink-0" aria-hidden />
              ) : (
                <CircleX className="size-3.5 shrink-0" aria-hidden />
              )}
              <span className="truncate">
                {d.last_test_ok ? "Passed" : "Failed"} · {relativeTime(d.last_test_at)}
              </span>
            </>
          ) : (
            "Never"
          )}
        </Metric>
      </dl>

      {error ? <ErrorCallout message={error} /> : null}
    </li>
  );
}

function ErrorCallout({ message }: { message: string }) {
  const [open, setOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  // First clause is usually the actionable summary; the rest is provider detail.
  const [summary] = message.split(/:\s/, 1);
  const long = message.length > 90;

  return (
    <div className="mt-4 rounded-xl bg-destructive-tint px-4 py-3 text-destructive sm:ml-[3.25rem]">
      <div className="flex items-start gap-2">
        <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-medium leading-5">{summary}</p>
          {open || !long ? (
            <p className="mt-1 max-h-40 overflow-y-auto break-words font-mono text-[11px] leading-relaxed opacity-90">
              {message}
            </p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          {long ? (
            <button
              type="button"
              aria-expanded={open}
              aria-label={open ? "Hide error details" : "Show error details"}
              onClick={() => setOpen((v) => !v)}
              className="flex size-7 items-center justify-center rounded-full outline-none transition-colors hover:bg-destructive/10 focus-visible:ring-2 focus-visible:ring-ring"
            >
              <ChevronDown className={cn("size-4 transition-transform", open && "rotate-180")} aria-hidden />
            </button>
          ) : null}
          <button
            type="button"
            aria-label={copied ? "Copied" : "Copy error"}
            title={copied ? "Copied" : "Copy error"}
            onClick={async () => {
              try {
                await copyToClipboard(message);
                setCopied(true);
                window.setTimeout(() => setCopied(false), 1600);
              } catch {
                /* clipboard denied */
              }
            }}
            className="flex size-7 items-center justify-center rounded-full outline-none transition-colors hover:bg-destructive/10 focus-visible:ring-2 focus-visible:ring-ring"
          >
            {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          </button>
        </div>
      </div>
    </div>
  );
}
