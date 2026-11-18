"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { ConnectionCredentialsPanel } from "@/components/services/connection-credentials";
import { Skeleton, SkeletonRegion } from "@/components/ui/skeleton";
import { ServiceDetailsDialog } from "@/components/services/service-details-dialog";
import type { RowPending } from "@/components/services/services-bulk-actions";
import {
  startStopTarget,
  stateBadgeVariant,
  type ServiceRow,
} from "@/lib/service-rows";
import { relativeTime } from "@/lib/overview-series";
import { formatBytes } from "@/lib/format-bytes";
import { cn } from "@/lib/utils";
import type { ServiceInstance } from "@/lib/api";
import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  Box,
  Cuboid,
  Database,
  HardDrive,
  Layers,
  Search,
  Server,
} from "lucide-react";
import * as React from "react";

const TYPE_ICONS = {
  postgres: Database,
  mysql: Server,
  mongodb: Cuboid,
  redis: Layers,
  rustfs: HardDrive,
  qdrant: Box,
  searxng: Search,
} as const;

const GRID =
  "grid grid-cols-[40px_minmax(150px,1fr)_80px_76px_92px_96px_104px_minmax(120px,1fr)_minmax(200px,auto)] items-center gap-2";

const ROW_HEIGHT = 56;

/** Centers row/header checkboxes in the 40px selection column. */
const SELECT_CELL = "flex items-center justify-center self-stretch";

type ServicesTableProps = {
  /** First load in flight: render skeleton rows instead of the empty state. */
  loading?: boolean;
  rows: ServiceRow[];
  totalCount: number;
  busyKey: string | null;
  filtered?: boolean;
  selected: Set<string>;
  onSelectedChange: React.Dispatch<React.SetStateAction<Set<string>>>;
  selectionDisabled?: boolean;
  bulkPending?: Record<string, RowPending>;
  /** Start (`true`) or stop (`false`) an instance. Resolves false if the request failed. */
  onToggle: (inst: ServiceInstance, enabled: boolean) => Promise<boolean>;
  onConfigure: (inst: ServiceInstance) => void;
  onTerminate: (inst: ServiceInstance) => Promise<boolean>;
  onClearFilters?: () => void;
};

const SKELETON_HEADERS = [
  "",
  "Service",
  "Type",
  "CPU",
  "Memory",
  "Last backup",
  "Status",
  "Endpoint",
  "",
];

/** Same grid + row height as the live table, so nothing jumps when data lands. */
export function ServicesTableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <SkeletonRegion
      label="Loading services"
      className="flex min-h-0 flex-col overflow-clip rounded-2xl bg-card shadow-rb-2"
    >
        <div
          className={cn(GRID, "border-b border-border bg-muted px-3")}
          style={{ height: ROW_HEIGHT }}
          aria-hidden
        >
        {SKELETON_HEADERS.map((h, i) => (
          <div
            key={i}
            className={cn("min-w-0", i === 0 && SELECT_CELL, i === 1 && "pl-[42px]")}
          >
            {i === 0 ? (
              <Skeleton className="size-5 rounded-full" />
            ) : (
              <span className="text-eyebrow text-muted-foreground">{h}</span>
            )}
          </div>
        ))}
      </div>
      {Array.from({ length: rows }, (_, i) => (
        <div
          key={i}
          className={cn(GRID, "border-b border-border px-3")}
          style={{ height: ROW_HEIGHT }}
          aria-hidden
        >
          <div className={SELECT_CELL}>
            <Skeleton className="size-5 rounded-full" />
          </div>
          <div className="flex items-center gap-2.5">
            <Skeleton className="size-8 shrink-0 rounded-full" />
            <Skeleton className="h-4 w-24" />
          </div>
          <Skeleton className="h-4 w-14" />
          <UsageSkeleton />
          <UsageSkeleton />
          <Skeleton className="h-4 w-14" />
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-4 w-40 max-w-full" />
          <div className="flex justify-end gap-1.5">
            <Skeleton className="h-8 w-[104px] rounded-full" />
            <Skeleton className="h-8 w-[96px] rounded-full" />
          </div>
        </div>
      ))}
      <div className="border-t border-border bg-muted px-4 py-3.5" aria-hidden>
        <Skeleton className="h-3 w-20" />
      </div>
    </SkeletonRegion>
  );
}

export function ServicesTable({
  loading = false,
  rows,
  totalCount,
  busyKey,
  filtered = false,
  selected,
  onSelectedChange,
  selectionDisabled = false,
  bulkPending,
  onToggle,
  onConfigure,
  onTerminate,
  onClearFilters,
}: ServicesTableProps) {
  const [sorting, setSorting] = React.useState<SortingState>([
    { id: "name", desc: false },
  ]);
  const [terminateTarget, setTerminateTarget] =
    React.useState<ServiceInstance | null>(null);
  const [terminateBusy, setTerminateBusy] = React.useState(false);

  React.useEffect(() => {
    const visible = new Set(rows.map((r) => r.rowKey));
    onSelectedChange((prev) => {
      const next = new Set([...prev].filter((k) => visible.has(k)));
      return next.size === prev.size ? prev : next;
    });
  }, [rows, onSelectedChange]);

  // Rows whose start/stop was just clicked. Kept disabled from the click until
  // the service actually leaves the state it was in (or the request fails), so
  // a slow status poll can never re-enable the button for a double click.
  const [pending, setPending] = React.useState<
    Record<string, { from: string; to: "start" | "stop" }>
  >({});
  const stalePending = Object.keys(pending).filter(
    (k) => rows.find((r) => r.rowKey === k)?.state !== pending[k].from,
  );
  if (stalePending.length > 0) {
    setPending((p) => {
      const next = { ...p };
      for (const k of stalePending) delete next[k];
      return next;
    });
  }

  const toggle = React.useCallback(
    (inst: ServiceInstance, from: string, to: "start" | "stop") => {
      const key = `${inst.type}-${inst.id}`;
      setPending((p) => ({ ...p, [key]: { from, to } }));
      void onToggle(inst, to === "start").then((ok) => {
        if (!ok) {
          setPending((p) => {
            const next = { ...p };
            delete next[key];
            return next;
          });
        }
      });
    },
    [onToggle],
  );

  const [detailsKey, setDetailsKey] = React.useState<string | null>(null);
  const detailsRow = rows.find((r) => r.rowKey === detailsKey) ?? null;

  const selectionLocked = selectionDisabled || busyKey !== null;

  const columns = React.useMemo<ColumnDef<ServiceRow>[]>(
    () => [
      {
        id: "select",
        header: () => null,
        enableSorting: false,
        cell: ({ row }) => (
          <Checkbox
            checked={selected.has(row.original.rowKey)}
            disabled={selectionLocked}
            onCheckedChange={(v) => {
              const key = row.original.rowKey;
              onSelectedChange((prev) => {
                const next = new Set(prev);
                if (v === true) next.add(key);
                else next.delete(key);
                return next;
              });
            }}
            aria-label={`Select ${row.original.instance.display_name || row.original.instance.id}`}
            onClick={(e) => e.stopPropagation()}
          />
        ),
      },
      {
        id: "name",
        accessorFn: (r) =>
          (r.instance.display_name || r.instance.id).toLowerCase(),
        header: "Service",
        cell: ({ row }) => {
          const inst = row.original.instance;
          const Icon =
            TYPE_ICONS[inst.type as keyof typeof TYPE_ICONS] ?? Database;
          const label = inst.display_name || inst.id;
          return (
            <div className="flex min-w-0 items-center gap-2.5">
              <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-broadcast-tint text-foreground">
                <Icon className="size-4" aria-hidden />
              </div>
              <button
                type="button"
                onClick={() => setDetailsKey(row.original.rowKey)}
                title="View details"
                className="min-w-0 truncate rounded-md text-left text-sm font-medium capitalize hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {label}
              </button>
            </div>
          );
        },
      },
      {
        id: "type",
        accessorFn: (r) => r.instance.type,
        header: "Type",
        cell: ({ getValue }) => (
          <span className="text-[13px] capitalize text-muted-foreground">
            {String(getValue())}
          </span>
        ),
      },
      {
        id: "cpu",
        accessorFn: (r) => usageSortValue(r, "cpu"),
        header: "CPU",
        sortUndefined: "last",
        cell: ({ row }) => <UsageCell row={row.original} kind="cpu" />,
      },
      {
        id: "memory",
        accessorFn: (r) => usageSortValue(r, "memory"),
        header: "Memory",
        sortUndefined: "last",
        cell: ({ row }) => <UsageCell row={row.original} kind="memory" />,
      },
      {
        id: "backup",
        accessorFn: (r) =>
          r.instance.deployment?.backup?.supported
            ? r.instance.deployment.backup.enabled
              ? r.lastBackupAt
                ? new Date(r.lastBackupAt).getTime()
                : 0
              : -1
            : -2,
        header: "Last backup",
        cell: ({ row }) => {
          const b = row.original.instance.deployment?.backup;
          const at = row.original.lastBackupAt;
          if (!b?.supported) {
            return (
              <span className="text-metadata text-muted-foreground">N/A</span>
            );
          }
          if (!b.enabled) {
            return (
              <span className="text-[13px] text-muted-foreground">Off</span>
            );
          }
          if (!at) {
            return (
              <span className="text-[13px] text-muted-foreground">Never</span>
            );
          }
          return (
            <span className="text-[13px]" title={new Date(at).toLocaleString()}>
              {relativeTime(at)}
            </span>
          );
        },
      },
      {
        id: "state",
        accessorFn: (r) => r.state,
        header: "Status",
        cell: ({ row }) => (
          <Badge
            variant={stateBadgeVariant(row.original.state)}
            className="-ml-2.5"
            title={row.original.statusHint}
          >
            {row.original.state}
          </Badge>
        ),
      },
      {
        id: "endpoint",
        header: "Remote connection",
        enableSorting: false,
        cell: ({ row }) => (
          <ConnectionCredentialsPanel
            connection={row.original.connection}
            enabled={row.original.instance.enabled}
            compact
          />
        ),
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        cell: ({ row }) => {
          const inst = row.original.instance;
          const state = row.original.state;
          const busy = busyKey === row.original.rowKey;
          const wait =
            pending[row.original.rowKey] ?? bulkPending?.[row.original.rowKey];
          // stopped -> Start; running/degraded -> Stop; anything in flight is disabled.
          const target = startStopTarget(state);
          const label = wait
            ? wait.to === "start"
              ? "Starting…"
              : "Stopping…"
            : state === "starting"
              ? "Starting…"
              : state === "stopping"
                ? "Stopping…"
                : target === "stop"
                  ? "Stop"
                  : "Start";
          return (
            <div className="flex flex-nowrap justify-end gap-1.5">
              <Button
                size="sm"
                variant={target === "stop" ? "outline" : "default"}
                className="w-[104px]"
                disabled={busy || !!wait || target === null}
                onClick={() => target && toggle(inst, state, target)}
              >
                {label}
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="w-[96px]"
                onClick={() => onConfigure(inst)}
              >
                Configure
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="w-[96px] text-destructive border-destructive/40 hover:bg-destructive-tint"
                disabled={busy || !!wait}
                onClick={() => setTerminateTarget(inst)}
              >
                Terminate
              </Button>
            </div>
          );
        },
      },
    ],
    [
      bulkPending,
      busyKey,
      onConfigure,
      onSelectedChange,
      pending,
      selected,
      selectionLocked,
      toggle,
    ],
  );

  const table = useReactTable({
    data: rows,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getRowId: (row) => row.rowKey,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const { rows: tableRows } = table.getRowModel();
  const visibleKeys = tableRows.map((r) => r.id);
  const allVisibleSelected =
    visibleKeys.length > 0 && visibleKeys.every((k) => selected.has(k));
  const someVisibleSelected = visibleKeys.some((k) => selected.has(k));

  const parentRef = React.useRef<HTMLDivElement>(null);

  const rowVirtualizer = useVirtualizer({
    count: tableRows.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 10,
  });

  if (loading) return <ServicesTableSkeleton />;

  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-border bg-card px-6 py-12 text-center shadow-rb-2">
        <p className="text-section-title">No services</p>
        <p className="mt-1 text-body text-muted-foreground">
          {totalCount === 0
            ? "Add an instance above to get started."
            : "No rows match your filters."}
        </p>
        {filtered && totalCount > 0 && onClearFilters && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={onClearFilters}
          >
            Clear filters
          </Button>
        )}
      </div>
    );
  }

  const terminateLabel = terminateTarget
    ? `${terminateTarget.type}/${terminateTarget.display_name || terminateTarget.id}`
    : "";

  return (
    <div className="flex min-h-0 flex-col overflow-clip rounded-2xl bg-card shadow-rb-2">
      <ServiceDetailsDialog
        row={detailsRow}
        onOpenChange={(o) => !o && setDetailsKey(null)}
        onConfigure={onConfigure}
      />
      <AlertDialog
        open={!!terminateTarget}
        onOpenChange={(o) => !o && !terminateBusy && setTerminateTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Terminate {terminateLabel}?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the instance from the appliance config, uninstalls its
              Helm release, and deletes local volume data (PVCs). It cannot be
              undone. Use <strong>Stop</strong> if you only want to scale down
              and keep data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={terminateBusy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={terminateBusy || !terminateTarget}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault();
                if (!terminateTarget) return;
                setTerminateBusy(true);
                void onTerminate(terminateTarget).then((ok) => {
                  setTerminateBusy(false);
                  if (ok) setTerminateTarget(null);
                });
              }}
            >
              {terminateBusy ? "Terminating…" : "Terminate permanently"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <div
        ref={parentRef}
        className="max-h-[min(640px,calc(100vh-18rem))] overflow-auto"
        role="region"
        aria-label="Services table"
      >
        <div
          className={cn(
            GRID,
            "sticky top-0 z-10 border-b border-border bg-muted px-3",
          )}
          style={{ height: ROW_HEIGHT }}
          role="row"
        >
          <div role="columnheader" className={SELECT_CELL}>
            <Checkbox
              checked={
                allVisibleSelected
                  ? true
                  : someVisibleSelected
                    ? "indeterminate"
                    : false
              }
              disabled={selectionLocked || visibleKeys.length === 0}
              onCheckedChange={(v) => {
                if (v === true) {
                  onSelectedChange(new Set(visibleKeys));
                } else {
                  onSelectedChange((prev) => {
                    const next = new Set(prev);
                    for (const k of visibleKeys) next.delete(k);
                    return next;
                  });
                }
              }}
              aria-label="Select all visible services"
            />
          </div>
          {table.getHeaderGroups().map((hg) =>
            hg.headers
              .filter((header) => header.column.id !== "select")
              .map((header) => (
              <div
                key={header.id}
                role="columnheader"
                className={cn(
                  "min-w-0",
                  header.column.id === "name" && "pl-[42px]",
                  header.column.id === "actions" && "text-right",
                )}
              >
                {header.isPlaceholder ? null : header.column.getCanSort() ? (
                  <button
                    type="button"
                    className={cn(
                      "text-eyebrow text-muted-foreground hover:text-foreground",
                      header.column.getIsSorted() && "text-foreground",
                    )}
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext(),
                    )}
                    {{
                      asc: " ↑",
                      desc: " ↓",
                    }[header.column.getIsSorted() as string] ?? null}
                  </button>
                ) : (
                  <span className="text-eyebrow text-muted-foreground">
                    {flexRender(
                      header.column.columnDef.header,
                      header.getContext(),
                    )}
                  </span>
                )}
              </div>
            )),
          )}
        </div>

        <div
          className="relative w-full"
          style={{ height: `${rowVirtualizer.getTotalSize()}px` }}
        >
          {rowVirtualizer.getVirtualItems().map((virtualRow) => {
            const row = tableRows[virtualRow.index];
            if (!row) return null;
            return (
              <div
                key={row.id}
                data-index={virtualRow.index}
                ref={rowVirtualizer.measureElement}
                className={cn(
                  GRID,
                  "absolute left-0 w-full overflow-hidden border-b border-border px-3 hover:bg-muted/40",
                )}
                style={{
                  height: ROW_HEIGHT,
                  transform: `translateY(${virtualRow.start}px)`,
                }}
                role="row"
              >
                {row.getVisibleCells().map((cell) => (
                  <div
                    key={cell.id}
                    role="cell"
                    className={cn(
                      "min-w-0",
                      cell.column.id === "select" && SELECT_CELL,
                      cell.column.id === "endpoint" && "overflow-hidden",
                    )}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>

      <div className="flex shrink-0 items-center border-t border-border bg-muted px-4 py-3.5 text-metadata text-muted-foreground">
        {tableRows.length === totalCount
          ? `${totalCount} instance${totalCount === 1 ? "" : "s"}`
          : `${tableRows.length} of ${totalCount} instances`}
      </div>
    </div>
  );
}

type UsageKind = "cpu" | "memory";

function usageMeter(row: ServiceRow, kind: UsageKind) {
  const m = row.vitals?.[kind];
  if (!row.vitals || !m || !m.known || m.not_running) return null;
  return m;
}

function usageSortValue(row: ServiceRow, kind: UsageKind): number | undefined {
  const m = usageMeter(row, kind);
  if (!m) return undefined;
  return kind === "memory" ? (m.used_bytes ?? m.percent) : m.percent;
}

function toneClass(pct: number) {
  return pct >= 85 ? "bg-destructive" : pct >= 65 ? "bg-warning" : "bg-feature";
}

/** One compact value with a hairline meter beneath it; "—" when not sampled. */
function UsageCell({ row, kind }: { row: ServiceRow; kind: UsageKind }) {
  const m = usageMeter(row, kind);
  if (!m) {
    return (
      <span
        className="text-[13px] text-muted-foreground"
        title={row.state === "running" ? "No sample yet" : "Not running"}
      >
        —
      </span>
    );
  }
  const pct = Math.min(100, Math.max(0, m.percent));
  const text =
    kind === "memory"
      ? formatBytes(m.used_bytes ?? 0)
      : m.percent > 0 && m.percent < 1
        ? "<1%"
        : `${m.percent < 10 ? m.percent.toFixed(1) : Math.round(m.percent)}%`;
  const title =
    kind === "memory" && m.total_bytes
      ? `${formatBytes(m.used_bytes ?? 0)} of ${formatBytes(m.total_bytes)} machine memory (${pct.toFixed(1)}%)`
      : `${pct.toFixed(1)}% of machine ${kind === "cpu" ? "CPU" : "memory"}`;
  return (
    <div className="w-full pr-2" title={title}>
      <span className="text-[13px] tabular-nums">{text}</span>
      <div
        className="mt-1 h-[3px] overflow-hidden rounded-full bg-muted"
        aria-hidden
      >
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-500",
            toneClass(pct),
          )}
          style={{ width: `${Math.max(pct, 3)}%` }}
        />
      </div>
    </div>
  );
}

function UsageSkeleton() {
  return (
    <div className="pr-2">
      <Skeleton className="h-3.5 w-10" />
      <Skeleton className="mt-1.5 h-[3px] w-full" />
    </div>
  );
}
