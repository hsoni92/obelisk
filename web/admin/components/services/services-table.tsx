"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/services/copy-button";
import { Skeleton, SkeletonRegion } from "@/components/ui/skeleton";
import { ServiceDetailsDialog } from "@/components/services/service-details-dialog";
import {
  connectionPreview,
  stateBadgeVariant,
  type ServiceRow,
} from "@/lib/service-rows";
import { relativeTime } from "@/lib/overview-series";
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
  "grid grid-cols-[minmax(160px,1fr)_88px_100px_112px_minmax(140px,1fr)_208px] items-center gap-2";

const ROW_HEIGHT = 56;

type ServicesTableProps = {
  /** First load in flight: render skeleton rows instead of the empty state. */
  loading?: boolean;
  rows: ServiceRow[];
  totalCount: number;
  busyKey: string | null;
  filtered?: boolean;
  /** Start (`true`) or stop (`false`) an instance. Resolves false if the request failed. */
  onToggle: (inst: ServiceInstance, enabled: boolean) => Promise<boolean>;
  onConfigure: (inst: ServiceInstance) => void;
  onClearFilters?: () => void;
};

const SKELETON_HEADERS = [
  "Service",
  "Type",
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
      className="flex min-h-0 flex-col rounded-2xl bg-card shadow-rb-2"
    >
      <div
        className={cn(GRID, "border-b border-border px-3 py-2.5")}
        aria-hidden
      >
        {SKELETON_HEADERS.map((h, i) => (
          <div key={i} className={cn("min-w-0", i === 0 && "pl-[42px]")}>
            <span className="text-eyebrow text-muted-foreground">{h}</span>
          </div>
        ))}
      </div>
      {Array.from({ length: rows }, (_, i) => (
        <div
          key={i}
          className={cn(GRID, "border-b border-border px-3 py-2")}
          style={{ height: ROW_HEIGHT }}
          aria-hidden
        >
          <div className="flex items-center gap-2.5">
            <Skeleton className="size-8 shrink-0 rounded-full" />
            <Skeleton className="h-4 w-24" />
          </div>
          <Skeleton className="h-4 w-14" />
          <Skeleton className="h-4 w-14" />
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-4 w-40 max-w-full" />
          <div className="flex justify-end gap-1.5">
            <Skeleton className="h-8 w-[104px] rounded-full" />
            <Skeleton className="h-8 w-[96px] rounded-full" />
          </div>
        </div>
      ))}
      <div className="px-4 py-2" aria-hidden>
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
  onToggle,
  onConfigure,
  onClearFilters,
}: ServicesTableProps) {
  const [sorting, setSorting] = React.useState<SortingState>([
    { id: "name", desc: false },
  ]);

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

  const columns = React.useMemo<ColumnDef<ServiceRow>[]>(
    () => [
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
          >
            {row.original.state}
          </Badge>
        ),
      },
      {
        id: "endpoint",
        header: "Endpoint",
        enableSorting: false,
        cell: ({ row }) => {
          const preview = connectionPreview(row.original.connection);
          const running = row.original.state === "running";
          if (!running) {
            return (
              <span className="text-metadata text-muted-foreground">
                Start to expose
              </span>
            );
          }
          if (!preview) {
            return (
              <span className="text-metadata text-muted-foreground">
                Pending connection…
              </span>
            );
          }
          return (
            <div className="flex min-w-0 items-center gap-1.5">
              <code
                className="min-w-0 flex-1 truncate font-mono text-[11px] text-foreground"
                title={preview}
              >
                {preview}
              </code>
              <CopyButton value={preview} />
            </div>
          );
        },
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        cell: ({ row }) => {
          const inst = row.original.instance;
          const state = row.original.state;
          const busy = busyKey === row.original.rowKey;
          const wait = pending[row.original.rowKey];
          // stopped -> Start; running/degraded -> Stop; anything in flight is disabled.
          const target: "start" | "stop" | null =
            state === "stopped"
              ? "start"
              : state === "running" || state === "degraded"
                ? "stop"
                : null;
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
            </div>
          );
        },
      },
    ],
    [busyKey, onConfigure, pending, toggle],
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

  return (
    <div className="flex min-h-0 flex-col rounded-2xl bg-card shadow-rb-2">
      <ServiceDetailsDialog
        row={detailsRow}
        onOpenChange={(o) => !o && setDetailsKey(null)}
        onConfigure={onConfigure}
      />
      <div
        ref={parentRef}
        className="max-h-[min(640px,calc(100vh-18rem))] overflow-auto"
        role="region"
        aria-label="Services table"
      >
        <div
          className={cn(
            GRID,
            "sticky top-0 z-10 border-b border-border bg-card px-3 py-2.5",
          )}
          role="row"
        >
          {table.getHeaderGroups().map((hg) =>
            hg.headers.map((header) => (
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
                  "absolute left-0 w-full border-b border-border px-3 py-2 hover:bg-muted/40",
                )}
                style={{
                  height: ROW_HEIGHT,
                  transform: `translateY(${virtualRow.start}px)`,
                }}
                role="row"
              >
                {row.getVisibleCells().map((cell) => (
                  <div key={cell.id} role="cell" className="min-w-0">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>

      <div className="border-t border-border px-4 py-2 text-metadata text-muted-foreground">
        {tableRows.length === totalCount
          ? `${totalCount} instance${totalCount === 1 ? "" : "s"}`
          : `${tableRows.length} of ${totalCount} instances`}
      </div>
    </div>
  );
}
