"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeading } from "@/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { SelectField } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { OverviewSkeleton } from "@/components/skeletons";
import {
  overviewPollOptions,
  rtkErrorMessage,
  rtkQueryLoadFailed,
  useGetBackupConfigQuery,
  useGetOverviewQuery,
} from "@/lib/api";
import { BackupOverview } from "@/components/overview/backup-overview";
import { ServiceVitals } from "@/components/overview/service-vitals";
import { summarizeBackup } from "@/lib/backup-summary";
import { formatBytes } from "@/lib/format-bytes";
import { buildTimeline, relativeTime } from "@/lib/overview-series";
import { cn } from "@/lib/utils";
import {
  AlertTriangle,
  ChartArea,
  CheckCircle2,
  Clock,
  DatabaseBackup,
  HardDrive,
  PieChart as PieChartIcon,
  ShieldAlert,
  type LucideIcon,
} from "lucide-react";
import { toast } from "@/lib/toast";
import { useEffect, useMemo, useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from "recharts";

const RANGE_OPTIONS = [
  { value: "1", label: "Last hour" },
  { value: "6", label: "Last 6 hours" },
  { value: "24", label: "Last 24 hours" },
];

const timelineConfig = {
  healthy: { label: "Running", color: "var(--chart-1)" },
  attention: { label: "Degraded / pending", color: "var(--destructive)" },
} satisfies ChartConfig;

const stateConfig = {
  running: { label: "Running", color: "var(--chart-1)" },
  starting: { label: "Starting", color: "var(--chart-3)" },
  stopping: { label: "Stopping", color: "var(--chart-3)" },
  pending: { label: "Pending", color: "var(--chart-3)" },
  degraded: { label: "Degraded", color: "var(--destructive)" },
  stopped: { label: "Stopped", color: "var(--chart-4)" },
} satisfies ChartConfig;

export function OverviewPanel({
  onOpenServices,
  onOpenDR,
}: {
  onOpenServices: (instanceKey?: string) => void;
  onOpenDR: () => void;
}) {
  const [hours, setHours] = useState(6);
  const { data, error, isLoading, isFetching, fulfilledTimeStamp } =
    useGetOverviewQuery({ hours }, overviewPollOptions);

  useEffect(() => {
    if (rtkQueryLoadFailed(error, data, isLoading, isFetching)) {
      toast.error("Could not load overview", rtkErrorMessage(error));
    }
  }, [error, data, isLoading, isFetching]);

  const { data: backupConfig } = useGetBackupConfigQuery(undefined, {
    pollingInterval: 8000,
  });
  const backup = useMemo(() => summarizeBackup(backupConfig), [backupConfig]);

  const components = useMemo(() => data?.components ?? [], [data?.components]);
  const summary = data?.summary;
  const total = components.length;

  const timeline = useMemo(
    () => buildTimeline(components, data?.history ?? [], hours),
    [components, data?.history, hours],
  );

  const donut = useMemo(() => {
    const n = (s: string) => components.filter((c) => c.state === s).length;
    const known = ["running", "starting", "stopping", "pending", "degraded"] as const;
    const rows = [
      ...known.map((s) => ({ state: s, count: n(s) })),
      {
        state: "stopped",
        count: components.filter((c) => !known.some((k) => k === c.state))
          .length,
      },
    ];
    return rows.map((r) => ({
      ...r,
      fill: `var(--color-${r.state})`,
      pct: total ? Math.round((r.count / total) * 100) : 0,
    }));
  }, [components, total]);

  if (isLoading) return <OverviewSkeleton />;

  if (error) {
    return (
      <p className="text-sm text-muted-foreground">
        Overview data is temporarily unavailable.
      </p>
    );
  }

  const pct = (n = 0) => (total ? `${Math.round((n / total) * 100)}%` : "—");
  const val = (n?: number) => n ?? 0;

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        {data?.reconcileMode === "off" && (
          <Badge variant="warning">reconcile off</Badge>
        )}
        {data?.profile && <Badge variant="info">{data.profile}</Badge>}
        {data?.role === "gateway" && (
          <Badge variant="warning">gateway</Badge>
        )}
        {data?.tailscale && <Badge variant="info">tailscale</Badge>}
        {data?.machineId && (
          <span className="font-mono text-[11px] text-muted-foreground">
            {data.machineId}
          </span>
        )}
        <span className="ml-auto hidden items-center gap-1 text-metadata text-muted-foreground sm:flex">
          <Clock className="size-3.5" />
          Updated{" "}
          {fulfilledTimeStamp
            ? new Date(fulfilledTimeStamp).toLocaleTimeString()
            : "…"}
        </span>
        {!data?.backup_ready && !isLoading && (
          <Button size="sm" variant="outline" onClick={onOpenDR}>
            Configure backup
          </Button>
        )}
        <SelectField
          aria-label="History range"
          className="w-36"
          value={String(hours)}
          onValueChange={(v) => setHours(Number(v))}
          options={RANGE_OPTIONS}
        />
      </div>

      {/* Verdict banner */}
      <Verdict
        loading={false}
        degraded={summary?.degraded ?? 0}
        pending={summary?.pending ?? 0}
        running={summary?.running ?? 0}
        total={total}
        backupTone={backup?.tone}
        backupDetail={backup?.detail}
        onOpenDR={onOpenDR}
        onOpenServices={() => onOpenServices()}
      />

      {/* KPIs */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Kpi
          icon={CheckCircle2}
          tone="success"
          label="Running"
          value={`${val(summary?.running)}/${total}`}
          pill={pct(summary?.running)}
          pillTone="success"
          hint="components healthy"
        />
        <Kpi
          icon={AlertTriangle}
          tone={summary?.degraded ? "live" : summary?.pending ? "warning" : "success"}
          label="Needs attention"
          value={val((summary?.degraded ?? 0) + (summary?.pending ?? 0))}
          pill={
            summary?.degraded
              ? `${summary.degraded} degraded`
              : summary?.pending
                ? `${summary.pending} pending`
                : "OK"
          }
          pillTone={
            summary?.degraded ? "live" : summary?.pending ? "warning" : "success"
          }
          hint={`${summary?.stopped ?? 0} stopped by design`}
        />
        <Kpi
          icon={backup?.tone === "success" ? DatabaseBackup : ShieldAlert}
          tone={backup?.tone ?? "default"}
          label="Backup status"
          value={backup?.label ?? "—"}
          loading={!backup}
          pill={
            backup
              ? `${backup.healthy}/${backup.enabled.length} dest`
              : ""
          }
          pillTone={backup?.tone ?? "default"}
          hint={
            backup?.lastBackupAt
              ? `last ${relativeTime(backup.lastBackupAt)}`
              : "no backup yet"
          }
        />
        <Kpi
          icon={HardDrive}
          tone="info"
          label="Backup size"
          value={backup ? (backup.totalBytes ? formatBytes(backup.totalBytes) : "0 B") : "—"}
          loading={!backup}
          pill={
            backup ? `${backup.totalObjects.toLocaleString()} obj` : ""
          }
          pillTone="default"
          hint="across enabled destinations"
        />
      </div>

      <ServiceVitals vitals={data?.vitals} />

      {/* Charts */}
      <div className="grid gap-3 lg:grid-cols-3">
        <Card className="overflow-hidden lg:col-span-2">
          <CardHeading
            icon={ChartArea}
            tone="blue"
            title="Availability timeline"
            description={`Components by state over the ${RANGE_OPTIONS.find(
              (o) => o.value === String(hours),
            )?.label.toLowerCase()}`}
            actions={
              <ChartLegend
                items={Object.entries(timelineConfig).map(([k, c]) => ({
                  label: c.label as string,
                  color: c.color,
                  key: k,
                }))}
              />
            }
          />
          <div className="p-4 pt-1">
          <ChartContainer
            config={timelineConfig}
            className="mt-3 aspect-auto h-56 w-full"
          >
            <AreaChart data={timeline} margin={{ left: -20, right: 4, top: 8 }}>
              <defs>
                {Object.keys(timelineConfig).map((k) => (
                  <linearGradient
                    key={k}
                    id={`fill-${k}`}
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="5%"
                      stopColor={`var(--color-${k})`}
                      stopOpacity={0.28}
                    />
                    <stop
                      offset="95%"
                      stopColor={`var(--color-${k})`}
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                ))}
              </defs>
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis
                dataKey="t"
                type="number"
                scale="time"
                domain={["dataMin", "dataMax"]}
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={48}
                tickFormatter={(t: number) =>
                  new Date(t).toLocaleTimeString([], {
                    hour: "2-digit",
                    minute: "2-digit",
                  })
                }
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                allowDecimals={false}
                domain={[0, Math.max(total, 1)]}
                tickMargin={4}
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    labelFormatter={(l) =>
                      new Date(Number(l)).toLocaleString([], {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      })
                    }
                  />
                }
              />
              <Area
                dataKey="attention"
                type="monotone"
                fill="url(#fill-attention)"
                stroke="var(--color-attention)"
                strokeWidth={2}
                isAnimationActive={false}
              />
              <Area
                dataKey="healthy"
                type="monotone"
                fill="url(#fill-healthy)"
                stroke="var(--color-healthy)"
                strokeWidth={2.5}
                isAnimationActive={false}
              />
            </AreaChart>
          </ChartContainer>
          </div>
        </Card>

        <Card className="flex flex-col overflow-hidden">
          <CardHeading
            icon={PieChartIcon}
            tone="green"
            title="Component health"
            description="Current distribution by state"
          />
          <div className="flex flex-1 flex-col p-4 pt-3">
          <div className="relative mx-auto mt-2 w-full max-w-[180px]">
            <ChartContainer
              config={stateConfig}
              className="aspect-square w-full"
            >
              <PieChart>
                <ChartTooltip
                  content={<ChartTooltipContent hideLabel nameKey="state" />}
                />
                <Pie
                  data={
                    total
                      ? donut
                      : [{ state: "stopped", count: 1, fill: "var(--chart-4)" }]
                  }
                  dataKey="count"
                  nameKey="state"
                  innerRadius="68%"
                  outerRadius="100%"
                  paddingAngle={total ? 2 : 0}
                  cornerRadius={6}
                  strokeWidth={0}
                  isAnimationActive={false}
                />
              </PieChart>
            </ChartContainer>
            <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-stat tabular-nums">{total}</span>
              <span className="text-eyebrow text-muted-foreground">
                Components
              </span>
            </div>
          </div>
          <ul className="mt-3 space-y-1.5">
            {donut.map((d) => (
              <li key={d.state} className="flex items-center gap-2 text-[13px]">
                <span
                  className="size-2 rounded-full"
                  style={{ background: `var(--color-${d.state})` }}
                />
                <span className="capitalize">{d.state}</span>
                <span className="ml-auto tabular-nums text-muted-foreground">
                  {d.count}
                </span>
                <span className="w-9 text-right font-medium tabular-nums">
                  {d.pct}%
                </span>
              </li>
            ))}
          </ul>
          </div>
        </Card>
      </div>

      {/* Backups */}
      <BackupOverview
        config={backupConfig}
        summary={backup}
        onOpenDR={onOpenDR}
      />
    </div>
  );
}

function ChartLegend({
  items,
}: {
  items: { key: string; label: string; color?: string }[];
}) {
  return (
    <ul className="flex flex-wrap justify-end gap-x-3 gap-y-1">
      {items.map((i) => (
        <li
          key={i.key}
          className="flex items-center gap-1.5 text-xs font-medium"
        >
          <span
            className="size-2 rounded-full"
            style={{ background: i.color }}
          />
          {i.label}
        </li>
      ))}
    </ul>
  );
}

const TONE_ICON: Record<string, string> = {
  success: "bg-success-tint text-success",
  warning: "bg-warning-tint text-warning",
  live: "bg-broadcast-tint text-foreground",
  info: "bg-link-tint text-link",
  default: "bg-muted text-muted-foreground",
};

function Kpi({
  icon: Icon,
  tone,
  label,
  value,
  pill,
  pillTone,
  hint,
  loading = false,
}: {
  icon: LucideIcon;
  tone: keyof typeof TONE_ICON;
  label: string;
  value: number | string;
  pill: string;
  pillTone: "success" | "warning" | "live" | "default";
  hint?: string;
  /** Value still being fetched: show placeholder blocks in place of value/pill/hint. */
  loading?: boolean;
}) {
  return (
    <Card className="p-4" aria-busy={loading || undefined}>
      <div className="flex items-start justify-between">
        <span
          className={cn(
            "flex size-8 items-center justify-center rounded-full",
            TONE_ICON[tone],
          )}
        >
          <Icon className="size-4" aria-hidden />
        </span>
        {loading ? (
          <Skeleton className="h-5 w-14 rounded-full" />
        ) : (
          <Badge variant={pillTone}>{pill}</Badge>
        )}
      </div>
      <p className="mt-3 text-eyebrow text-muted-foreground">{label}</p>
      {loading ? (
        <Skeleton className="mt-1 h-9 w-24 rounded-xl" />
      ) : (
        <p className="text-stat tabular-nums">{value}</p>
      )}
      {loading ? (
        <Skeleton className="mt-2 h-3 w-32" />
      ) : hint && (
        <p className="mt-0.5 truncate text-metadata text-muted-foreground">
          {hint}
        </p>
      )}
    </Card>
  );
}

function Verdict({
  loading,
  degraded,
  pending,
  running,
  total,
  backupTone,
  backupDetail,
  onOpenDR,
  onOpenServices,
}: {
  loading: boolean;
  degraded: number;
  pending: number;
  running: number;
  total: number;
  backupTone?: "success" | "warning" | "live" | "default";
  backupDetail?: string;
  onOpenDR: () => void;
  onOpenServices: () => void;
}) {
  if (loading) return null;
  const issues: string[] = [];
  if (degraded) issues.push(`${degraded} component${degraded > 1 ? "s" : ""} degraded`);
  if (pending) issues.push(`${pending} pending`);
  const backupBad = backupTone === "warning" || backupTone === "live";
  if (backupBad && backupDetail) issues.push(backupDetail);

  const bad = degraded > 0 || backupTone === "live";
  const ok = issues.length === 0;
  const Icon = ok ? CheckCircle2 : AlertTriangle;
  return (
    <div
      role="status"
      className={cn(
        "flex flex-wrap items-center gap-3 rounded-xl border px-4 py-3",
        ok && "border-success/30 bg-success-tint text-success",
        !ok && bad && "border-destructive/30 bg-destructive-tint text-destructive",
        !ok && !bad && "border-warning/30 bg-warning-tint text-warning",
      )}
    >
      <Icon className="size-5 shrink-0" aria-hidden />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold">
          {ok
            ? "All systems operational"
            : bad
              ? "Action needed"
              : "Heads up"}
        </p>
        <p className="text-metadata opacity-90">
          {ok
            ? `${running} of ${total} components running · backups healthy`
            : issues.join(" · ")}
        </p>
      </div>
      {!ok && (degraded > 0 || pending > 0) && (
        <Button size="sm" variant="outline" onClick={onOpenServices}>
          View services
        </Button>
      )}
      {backupBad && (
        <Button size="sm" variant="outline" onClick={onOpenDR}>
          Fix backups
        </Button>
      )}
    </div>
  );
}
