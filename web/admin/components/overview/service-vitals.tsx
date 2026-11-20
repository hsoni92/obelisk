"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Activity,
  Cpu,
  HardDrive,
  ListTree,
  MemoryStick,
  Search,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { Card, CardHeading } from "@/components/ui/card";
import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { formatBytes } from "@/lib/format-bytes";
import type { VitalsMeter, VitalsRow, VitalsSnapshot } from "@/lib/types";
import { Area, AreaChart, YAxis } from "recharts";

const seriesConfig = {
  pct: { label: "Share of machine", color: "var(--chart-1)" },
} satisfies ChartConfig;

export function ServiceVitals({ vitals }: { vitals?: VitalsSnapshot }) {
  if (!vitals) return null;

  const updated = vitals.collected_at
    ? new Date(vitals.collected_at).toLocaleTimeString()
    : "…";

  return (
    <Card className="overflow-hidden">
      <CardHeading
        icon={Activity}
        tone="lime"
        title="Service vitals"
        description="Live machine load, sampled about once a minute."
        actions={<VitalsMeta vitals={vitals} updated={updated} />}
      />
      <div className="p-5 pt-0 sm:p-6 sm:pt-0">
      <MachineTotals vitals={vitals} />

      <ServiceVitalsSummary vitals={vitals} />
      </div>
    </Card>
  );
}

function VitalsMeta({
  vitals,
  updated,
}: {
  vitals: VitalsSnapshot;
  updated: string;
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-metadata text-muted-foreground">
        Slot {updated}
      </span>
      {vitals.collecting && !vitals.collected_at ? (
        <Badge variant="warning">collecting</Badge>
      ) : null}
      {vitals.stale ? <Badge variant="warning">stale</Badge> : null}
      {vitals.scarce_resource ? (
        <Badge variant="info">short on {vitals.scarce_resource}</Badge>
      ) : null}
    </div>
  );
}

function MachineTotals({ vitals }: { vitals: VitalsSnapshot }) {
  const m = vitals.machine;
  const sampled =
    Boolean(vitals.collected_at) &&
    !vitals.collecting &&
  (m.memory_total_bytes > 0 || m.cpu_percent > 0 || m.disk_total_bytes > 0);
  return (
    <div className="mt-5 grid gap-4 md:grid-cols-3">
      <MachineMeter
        label="Memory"
        icon={MemoryStick}
        id="mem"
        meter={{
          percent: pct(m.memory_used_bytes, m.memory_total_bytes),
          used_bytes: m.memory_used_bytes,
          total_bytes: m.memory_total_bytes,
          known: sampled && m.memory_total_bytes > 0,
          series: vitals.machine.memory_series,
        }}
      />
      <MachineMeter
        label="CPU"
        icon={Cpu}
        id="cpu"
        meter={{
          percent: m.cpu_percent,
          known: sampled,
          series: vitals.machine.cpu_series,
        }}
      />
      <MachineMeter
        label="Disk (est.)"
        icon={HardDrive}
        id="disk"
        meter={{
          percent: pct(m.disk_used_bytes, m.disk_total_bytes),
          used_bytes: m.disk_used_bytes,
          total_bytes: m.disk_total_bytes,
          known: sampled && m.disk_total_bytes > 0,
          series: vitals.machine.disk_series,
        }}
      />
    </div>
  );
}

/** "2.8 GB" -> ["2.8", "GB"] so the number can be large and the unit small. */
function splitBytes(bytes?: number): [string, string] {
  const t = formatVitalsBytes(bytes);
  const i = t.indexOf(" ");
  return i < 0 ? [t, ""] : [t.slice(0, i), t.slice(i + 1)];
}

function MachineMeter({
  id,
  label,
  icon: Icon,
  meter,
}: {
  id: string;
  label: string;
  icon: LucideIcon;
  meter: VitalsMeter & { series?: { at: string; percent: number }[] };
}) {
  const p = meter.known ? clampPct(meter.percent) : 0;
  const hot = p >= 85;
  const warm = p >= 65;
  const stroke = hot
    ? "var(--destructive)"
    : warm
      ? "var(--warning)"
      : "var(--chart-1)";

  const [big, unit] = !meter.known
    ? ["—", ""]
    : meter.total_bytes
      ? splitBytes(meter.used_bytes)
      : [String(Math.round(meter.percent)), "%"];
  const sub = !meter.known
    ? "unavailable"
    : meter.total_bytes
      ? `of ${formatBytes(meter.total_bytes)}`
      : "of machine";

  const series = (meter.series ?? [])
    .filter((pt) => pt.percent >= 0)
    .map((pt) => ({
      t: new Date(pt.at).getTime(),
      pct: Math.round(pt.percent * 10) / 10,
    }));

  return (
    <div className="relative flex min-h-[184px] flex-col overflow-clip rounded-2xl bg-muted p-5">
      <div className="flex items-center justify-between gap-2">
        <p className="flex items-center gap-2 text-eyebrow text-muted-foreground">
          <Icon className="size-3.5" aria-hidden />
          {label}
        </p>
        {meter.known && (
          <span
            className={cn(
              "rounded-full px-2.5 py-0.5 text-[11px] tabular-nums",
              hot
                ? "bg-destructive-tint text-destructive"
                : warm
                  ? "bg-warning-tint text-warning"
                  : "bg-card text-foreground shadow-rb-1",
            )}
          >
            {Math.round(meter.percent)}%
          </span>
        )}
      </div>

      <div className="mt-3 flex items-baseline gap-1.5">
        <span className="text-stat tabular-nums">{big}</span>
        {unit && (
          <span className="text-base font-normal text-muted-foreground">
            {unit}
          </span>
        )}
      </div>
      <p className="text-metadata text-muted-foreground">{sub}</p>

      <div className="mt-auto pt-4">
        {series.length > 1 ? (
          <ChartContainer
            config={seriesConfig}
            className="-mx-5 -mb-5 h-14 w-[calc(100%+2.5rem)]"
          >
            <AreaChart
              data={series}
              margin={{ left: 0, right: 0, top: 4, bottom: 0 }}
            >
              <defs>
                <linearGradient id={`vit-${id}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={stroke} stopOpacity={0.35} />
                  <stop offset="100%" stopColor={stroke} stopOpacity={0} />
                </linearGradient>
              </defs>
              <YAxis hide domain={[0, 100]} />
              <Area
                type="monotone"
                dataKey="pct"
                stroke={stroke}
                strokeWidth={1.75}
                fill={`url(#vit-${id})`}
                dot={false}
                isAnimationActive={false}
              />
            </AreaChart>
          </ChartContainer>
        ) : (
          <div
            className="h-1.5 overflow-hidden rounded-full bg-card"
            aria-hidden
          >
            <div
              className={cn(
                "h-full rounded-full transition-[width] duration-500",
                hot ? "bg-destructive" : warm ? "bg-warning" : "bg-feature",
              )}
              style={{ width: `${p}%` }}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function pct(used: number, total: number) {
  if (!total) return 0;
  return (100 * used) / total;
}

function clampPct(n: number) {
  if (n < 0) return 0;
  if (n > 100) return 100;
  return n;
}

/** Used bytes: 0 is valid; only null/undefined is unknown. */
function formatVitalsBytes(bytes?: number) {
  if (bytes == null) return "—";
  if (bytes === 0) return "0 B";
  return formatBytes(bytes);
}

type SortKey = "service" | "state" | "memory" | "cpu" | "disk";

function sortValue(row: VitalsRow, key: SortKey): number | string {
  const m = key === "memory" ? row.memory : key === "cpu" ? row.cpu : row.disk;
  switch (key) {
    case "service":
      return row.name.toLowerCase();
    case "state":
      return row.running ? 1 : 0;
    case "cpu":
      return m.known && !m.not_running ? m.percent : -1;
    default:
      return m.known && !(m.not_running && key !== "disk")
        ? (m.used_bytes ?? m.percent)
        : -1;
  }
}

/** Compact card footer: top consumers + CTA to the full sortable table. */
function ServiceVitalsSummary({ vitals }: { vitals: VitalsSnapshot }) {
  const [open, setOpen] = React.useState(false);
  const rows = vitals.rows ?? [];
  const running = rows.filter((r) => r.running).length;
  const top = rows.slice(0, 3);

  return (
    <>
      <div className="mt-4 flex flex-wrap items-center gap-3 rounded-xl bg-muted/60 p-3">
        <span className="flex size-9 items-center justify-center rounded-full bg-card text-muted-foreground shadow-rb-1">
          <ListTree className="size-4" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">
            {rows.length} services
            <span className="font-normal text-muted-foreground">
              {" "}
              · {running} running
            </span>
          </p>
          {top.length > 0 && (
            <p className="truncate text-metadata text-muted-foreground">
              Top by {vitals.scarce_resource ?? "memory"}:{" "}
              {top.map((r) => r.name).join(", ")}
            </p>
          )}
        </div>
        <Button
          variant="ink"
          size="sm"
          onClick={() => setOpen(true)}
          disabled={rows.length === 0}
        >
          View all services
        </Button>
      </div>
      <ServiceVitalsDialog open={open} onOpenChange={setOpen} vitals={vitals} />
    </>
  );
}

function ServiceVitalsDialog({
  open,
  onOpenChange,
  vitals,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  vitals: VitalsSnapshot;
}) {
  const [sort, setSort] = React.useState<{ key: SortKey; dir: "asc" | "desc" }>(
    { key: "memory", dir: "desc" },
  );
  const [query, setQuery] = React.useState("");

  const baseRows = vitals.rows ?? [];

  const rows = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    const filtered = q
      ? baseRows.filter(
          (r) =>
            r.name.toLowerCase().includes(q) || r.key.toLowerCase().includes(q),
        )
      : baseRows;
    const dir = sort.dir === "asc" ? 1 : -1;
    return [...filtered].sort((a, b) => {
      const av = sortValue(a, sort.key);
      const bv = sortValue(b, sort.key);
      const c =
        typeof av === "string" && typeof bv === "string"
          ? av.localeCompare(bv)
          : (av as number) - (bv as number);
      return c * dir || a.name.localeCompare(b.name);
    });
  }, [baseRows, sort, query]);

  function toggle(key: SortKey) {
    setSort((s) =>
      s.key === key
        ? { key, dir: s.dir === "asc" ? "desc" : "asc" }
        : { key, dir: key === "service" ? "asc" : "desc" },
    );
  }

  const cols: { key: SortKey; label: string; className?: string }[] = [
    { key: "service", label: "Service" },
    { key: "state", label: "State", className: "w-24" },
    { key: "memory", label: "Memory", className: "w-48" },
    { key: "cpu", label: "CPU", className: "w-36" },
    { key: "disk", label: "Disk", className: "w-48" },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Service vitals</DialogTitle>
          <DialogDescription>
            Latest samples per service
            {vitals.scarce_resource
              ? ` · ranked by ${vitals.scarce_resource}`
              : ""}
            . Click a column to sort.
          </DialogDescription>
        </DialogHeader>
        <div className="shrink-0 border-b border-border px-6 py-3">
          <div className="flex h-9 max-w-xs items-center gap-2 rounded-full bg-input px-3.5 text-muted-foreground focus-within:ring-2 focus-within:ring-ring/60">
            <Search className="size-3.5 shrink-0" aria-hidden />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Filter services…"
              aria-label="Filter services"
              className="flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
            />
          </div>
        </div>
        <DialogBody className="space-y-0 px-0 py-0">
          <table className="w-full border-separate border-spacing-0 text-sm">
            <thead>
              <tr>
                {cols.map((c) => {
                  const active = sort.key === c.key;
                  const Icon = !active
                    ? ArrowUpDown
                    : sort.dir === "asc"
                      ? ArrowUp
                      : ArrowDown;
                  return (
                    <th
                      key={c.key}
                      scope="col"
                      aria-sort={
                        active
                          ? sort.dir === "asc"
                            ? "ascending"
                            : "descending"
                          : "none"
                      }
                      className={cn(
                        "sticky top-0 z-10 border-b border-border bg-card px-3 py-2 text-left first:pl-6 last:pr-6",
                        c.className,
                      )}
                    >
                      <button
                        type="button"
                        onClick={() => toggle(c.key)}
                        className={cn(
                          "-mx-1.5 inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-eyebrow transition-colors hover:text-foreground",
                          active ? "text-foreground" : "text-muted-foreground",
                        )}
                      >
                        {c.label}
                        <Icon
                          className={cn("size-3", !active && "opacity-40")}
                          aria-hidden
                        />
                      </button>
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.key} className="group">
                  <td className="border-b border-border/60 px-3 py-2.5 pl-6 group-hover:bg-muted/50">
                    <div className="flex items-baseline gap-2">
                      <span className="font-medium">{row.name}</span>
                      <span className="truncate font-mono text-[11px] text-muted-foreground">
                        {row.key}
                      </span>
                    </div>
                  </td>
                  <td className="border-b border-border/60 px-3 py-2.5 group-hover:bg-muted/50">
                    <Badge variant={row.running ? "success" : "default"}>
                      {row.running ? "running" : "stopped"}
                    </Badge>
                  </td>
                  <td className="border-b border-border/60 px-3 py-2.5 group-hover:bg-muted/50">
                    <Cell meter={row.memory} />
                  </td>
                  <td className="border-b border-border/60 px-3 py-2.5 group-hover:bg-muted/50">
                    <Cell meter={row.cpu} />
                  </td>
                  <td className="border-b border-border/60 px-3 py-2.5 pr-6 group-hover:bg-muted/50">
                    <Cell meter={row.disk} diskCell />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length === 0 && (
            <p className="px-6 py-10 text-center text-sm text-muted-foreground">
              No services match “{query}”.
            </p>
          )}
        </DialogBody>
        <DialogFooter className="justify-between">
          <span className="text-metadata text-muted-foreground">
            {rows.length} of {vitals.rows.length} services
          </span>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Cell({ meter, diskCell }: { meter: VitalsMeter; diskCell?: boolean }) {
  if (!meter.known || (meter.not_running && !diskCell)) {
    return (
      <span className="text-muted-foreground">{meter.known ? "—" : "n/a"}</span>
    );
  }
  const p = clampPct(meter.percent);
  return (
    <div className="flex items-center gap-2.5">
      <span className="w-[4.5rem] shrink-0 tabular-nums">
        {meter.total_bytes
          ? formatVitalsBytes(meter.used_bytes)
          : `${Math.round(meter.percent)}%`}
      </span>
      <div
        className="h-1 w-14 shrink-0 overflow-hidden rounded-full bg-muted"
        aria-hidden
      >
        <div
          className={cn(
            "h-full rounded-full",
            p >= 85 ? "bg-destructive" : p >= 65 ? "bg-warning" : "bg-feature",
          )}
          style={{ width: `${Math.max(p, 2)}%` }}
        />
      </div>
      {meter.total_bytes ? (
        <span className="text-metadata tabular-nums text-muted-foreground">
          {Math.round(meter.percent)}%
        </span>
      ) : null}
    </div>
  );
}
