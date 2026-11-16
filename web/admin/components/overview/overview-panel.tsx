"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { SelectField } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { OverviewSkeleton } from "@/components/skeletons";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  overviewPollOptions,
  rtkErrorMessage,
  useGetBackupConfigQuery,
  useGetOverviewQuery,
} from "@/lib/api";
import { BackupOverview } from "@/components/overview/backup-overview";
import { summarizeBackup } from "@/lib/backup-summary";
import { formatBytes } from "@/lib/format-bytes";
import { buildTimeline, relativeTime, severity } from "@/lib/overview-series";
import { stateBadgeVariant } from "@/lib/service-rows";
import { cn } from "@/lib/utils";
import {
  AlertTriangle,
  ArrowUpRight,
  CheckCircle2,
  Clock,
  DatabaseBackup,
  HardDrive,
  ShieldAlert,
  type LucideIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
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
  healthy: { label: "Running", color: "#22c55e" },
  attention: { label: "Degraded / pending", color: "#ef4444" },
} satisfies ChartConfig;

const stateConfig = {
  running: { label: "Running", color: "#22c55e" },
  starting: { label: "Starting", color: "#eab308" },
  stopping: { label: "Stopping", color: "#eab308" },
  pending: { label: "Pending", color: "#eab308" },
  degraded: { label: "Degraded", color: "#ef4444" },
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
  const [view, setView] = useState<"attention" | "all" | null>(null);
  const [showAllEvents, setShowAllEvents] = useState(false);
  const { data, error, isLoading, fulfilledTimeStamp } = useGetOverviewQuery(
    { hours },
    overviewPollOptions,
  );

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

  const attention = useMemo(
    () => components.filter((c) => c.state !== "running" && c.desired),
    [components],
  );
  const activeView = view ?? (attention.length > 0 ? "attention" : "all");
  const rows = useMemo(
    () =>
      [...(activeView === "attention" ? attention : components)].sort(
        (a, b) =>
          severity(a.state) - severity(b.state) || a.name.localeCompare(b.name),
      ),
    [activeView, attention, components],
  );

  const events = useMemo(
    () =>
      [...(data?.history ?? [])].sort(
        (a, b) => new Date(b.at).getTime() - new Date(a.at).getTime(),
      ),
    [data?.history],
  );
  const shownEvents = showAllEvents ? events : events.slice(0, 8);

  if (isLoading) return <OverviewSkeleton />;

  if (error) {
    return (
      <p className="inline-block rounded-full bg-broadcast-tint px-3 py-1 text-xs font-medium text-foreground">
        {rtkErrorMessage(error)}
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
        onShowAttention={() => setView("attention")}
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

      {/* Charts */}
      <div className="grid gap-3 lg:grid-cols-3">
        <Card className="p-4 lg:col-span-2">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h2 className="text-section-title">Availability timeline</h2>
              <p className="text-metadata text-muted-foreground">
                Components by state over the{" "}
                {RANGE_OPTIONS.find(
                  (o) => o.value === String(hours),
                )?.label.toLowerCase()}
              </p>
            </div>
            <ChartLegend
              items={Object.entries(timelineConfig).map(([k, c]) => ({
                label: c.label as string,
                color: c.color,
                key: k,
              }))}
            />
          </div>
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
        </Card>

        <Card className="flex flex-col p-4">
          <h2 className="text-section-title">Component health</h2>
          <p className="text-metadata text-muted-foreground">
            Current distribution by state
          </p>
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
        </Card>
      </div>

      {/* Backups */}
      <BackupOverview
        config={backupConfig}
        summary={backup}
        onOpenDR={onOpenDR}
      />

      {/* Components */}
      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-2 p-4 pb-3">
          <div>
            <h2 className="text-section-title">Components</h2>
            <p className="text-metadata text-muted-foreground">
              Sorted by severity — problems first
            </p>
          </div>
          <Tabs
            value={activeView}
            onValueChange={(v) => setView(v as "attention" | "all")}
          >
            <TabsList className="bg-transparent p-0">
              <TabsTrigger value="attention">
                Needs attention
                <span className="ml-1.5 tabular-nums opacity-70">
                  {attention.length}
                </span>
              </TabsTrigger>
              <TabsTrigger value="all">
                All
                <span className="ml-1.5 tabular-nums opacity-70">{total}</span>
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
        <div className="max-h-80 overflow-auto border-t border-border">
          <table className="w-full text-[13px]">
            <thead className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <tr className="text-left text-eyebrow text-muted-foreground">
                <th className="px-4 py-2 font-semibold">Component</th>
                <th className="px-4 py-2 font-semibold">Status</th>
                <th className="hidden px-4 py-2 font-semibold sm:table-cell">
                  Pods
                </th>
                <th className="hidden px-4 py-2 font-semibold md:table-cell">
                  Helm
                </th>
                <th className="w-10 px-4 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr
                  key={c.key}
                  className="border-t border-border hover:bg-muted/40"
                >
                  <td className="px-4 py-2">
                    <p className="font-semibold">{c.name}</p>
                    <p className="text-metadata text-muted-foreground">
                      {c.kind === "platform"
                        ? "platform"
                        : `${c.type} · ${c.instance_id}`}
                      {c.reason ? ` — ${c.reason}` : ""}
                    </p>
                  </td>
                  <td className="px-4 py-2">
                    <Badge variant={stateBadgeVariant(c.state)}>
                      {c.state}
                    </Badge>
                  </td>
                  <td className="hidden px-4 py-2 sm:table-cell">
                    {c.pods_total ? (
                      <div className="flex w-24 items-center gap-2">
                        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                          <div
                            className={cn(
                              "h-full rounded-full",
                              c.pods_ready === c.pods_total
                                ? "bg-success"
                                : "bg-warning",
                            )}
                            style={{
                              width: `${((c.pods_ready ?? 0) / c.pods_total) * 100}%`,
                            }}
                          />
                        </div>
                        <span className="tabular-nums text-metadata text-muted-foreground">
                          {c.pods_ready}/{c.pods_total}
                        </span>
                      </div>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="hidden px-4 py-2 text-metadata text-muted-foreground md:table-cell">
                    {c.helm_status ?? "—"}
                    {c.revision ? ` · r${c.revision}` : ""}
                  </td>
                  <td className="px-4 py-2 text-right">
                    {c.kind === "instance" && (
                      <button
                        type="button"
                        aria-label={`Open ${c.name} in Services`}
                        title="Open in Services"
                        onClick={() => onOpenServices(c.key)}
                        className="rounded-full p-1 text-muted-foreground hover:bg-surface hover:text-foreground"
                      >
                        <ArrowUpRight className="size-4" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {!isLoading && rows.length === 0 && (
                <tr>
                  <td
                    colSpan={5}
                    className="px-4 py-8 text-center text-muted-foreground"
                  >
                    {activeView === "attention"
                      ? "Everything is healthy — nothing needs attention."
                      : "No components."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Activity */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between p-4 pb-3">
          <div>
            <h2 className="text-section-title">Recent activity</h2>
            <p className="text-metadata text-muted-foreground">
              State transitions · {events.length} in window
            </p>
          </div>
          {events.length > 8 && (
            <Button variant="link" onClick={() => setShowAllEvents((s) => !s)}>
              {showAllEvents ? "Show less" : `View all (${events.length})`}
            </Button>
          )}
        </div>
        <div className="max-h-80 overflow-auto border-t border-border">
          <table className="w-full text-[13px]">
            <thead className="sticky top-0 z-10 bg-muted/90 backdrop-blur">
              <tr className="text-left text-eyebrow text-muted-foreground">
                <th className="px-4 py-2 font-semibold">When</th>
                <th className="px-4 py-2 font-semibold">Component</th>
                <th className="px-4 py-2 font-semibold">Status</th>
                <th className="hidden px-4 py-2 font-semibold md:table-cell">
                  Detail
                </th>
              </tr>
            </thead>
            <tbody>
              {shownEvents.map((ev, i) => (
                <tr
                  key={`${ev.at}-${ev.key}-${i}`}
                  className="border-t border-border hover:bg-muted/40"
                >
                  <td
                    className="whitespace-nowrap px-4 py-2 text-muted-foreground"
                    title={new Date(ev.at).toLocaleString()}
                  >
                    {relativeTime(ev.at)}
                  </td>
                  <td className="px-4 py-2 font-medium">{ev.key}</td>
                  <td className="px-4 py-2">
                    <Badge variant={stateBadgeVariant(ev.state)}>
                      {ev.state}
                    </Badge>
                  </td>
                  <td className="hidden max-w-xs truncate px-4 py-2 text-metadata text-muted-foreground md:table-cell">
                    {ev.detail ?? "—"}
                  </td>
                </tr>
              ))}
              {events.length === 0 && (
                <tr>
                  <td
                    colSpan={4}
                    className="px-4 py-8 text-center text-muted-foreground"
                  >
                    No transitions in this window. Changes appear as component
                    state shifts.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Card>
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
  onShowAttention,
}: {
  loading: boolean;
  degraded: number;
  pending: number;
  running: number;
  total: number;
  backupTone?: "success" | "warning" | "live" | "default";
  backupDetail?: string;
  onOpenDR: () => void;
  onShowAttention: () => void;
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
        <Button size="sm" variant="outline" onClick={onShowAttention}>
          Show components
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
