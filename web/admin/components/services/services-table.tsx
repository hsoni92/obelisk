"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/services/copy-button";
import {
  connectionPreview,
  stateBadgeVariant,
  type ServiceRow,
} from "@/lib/service-rows";
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
  "grid grid-cols-[minmax(160px,1fr)_88px_100px_112px_minmax(140px,1fr)_minmax(200px,auto)] items-center gap-2";

const ROW_HEIGHT = 56;

type ServicesTableProps = {
  rows: ServiceRow[];
  totalCount: number;
  busyKey: string | null;
  connectionNotes?: string;
  filtered?: boolean;
  onStart: (inst: ServiceInstance) => void;
  onStop: (inst: ServiceInstance) => void;
  onConfigure: (inst: ServiceInstance) => void;
  onClearFilters?: () => void;
};

export function ServicesTable({
  rows,
  totalCount,
  busyKey,
  connectionNotes,
  filtered = false,
  onStart,
  onStop,
  onConfigure,
  onClearFilters,
}: ServicesTableProps) {
  const [sorting, setSorting] = React.useState<SortingState>([
    { id: "name", desc: false },
  ]);

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
              <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-broadcast-tint text-broadcast">
                <Icon className="size-4" aria-hidden />
              </div>
              <div className="min-w-0">
                <p className="truncate text-[13px] font-semibold capitalize">
                  {label}
                </p>
                <p className="truncate text-eyebrow text-muted-foreground">
                  {inst.deployment?.release ?? inst.type} · {inst.type} ·{" "}
                  {inst.id}
                </p>
              </div>
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
              ? "on"
              : "off"
            : "na",
        header: "Backup",
        cell: ({ row }) => {
          const b = row.original.instance.deployment?.backup;
          if (!b?.supported) {
            return (
              <span className="text-metadata text-muted-foreground">N/A</span>
            );
          }
          const label = !b.enabled
            ? "off"
            : b.policy?.overridden
              ? `${b.method ?? "on"} · custom`
              : (b.method ?? "on");
          return (
            <Badge
              variant={b.enabled ? "success" : "default"}
              title={
                b.enabled && b.policy
                  ? `RF ${b.policy.replication_factor} · ${b.policy.schedule}`
                  : undefined
              }
            >
              {label}
            </Badge>
          );
        },
      },
      {
        id: "state",
        accessorFn: (r) => r.state,
        header: "Status",
        cell: ({ row }) => (
          <Badge variant={stateBadgeVariant(row.original.state)}>
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
          const running = state === "running";
          const pending = state === "pending";
          const loading = state === "loading";
          const busy = busyKey === row.original.rowKey;
          return (
            <div className="flex flex-wrap justify-end gap-1.5">
              <Button
                size="sm"
                disabled={busy || running || pending || loading}
                onClick={() => onStart(inst)}
              >
                {busy ? "…" : "Start"}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={busy || !running || loading}
                onClick={() => onStop(inst)}
              >
                Stop
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => onConfigure(inst)}
              >
                Configure
              </Button>
            </div>
          );
        },
      },
    ],
    [busyKey, onConfigure, onStart, onStop],
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
    <div className="flex min-h-0 flex-col rounded-xl border border-border bg-card shadow-rb-2">
      {connectionNotes && (
        <p className="border-b border-border px-4 py-2 text-metadata text-muted-foreground">
          {connectionNotes}
        </p>
      )}

      <div className={cn(GRID, "border-b border-border px-3 py-2")} role="row">
        {table.getHeaderGroups().map((hg) =>
          hg.headers.map((header) => (
            <div key={header.id} role="columnheader" className="min-w-0">
              {header.isPlaceholder ? null : header.column.getCanSort() ? (
                <button
                  type="button"
                  className={cn(
                    "text-eyebrow text-muted-foreground hover:text-foreground",
                    header.column.getIsSorted() && "text-broadcast",
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
        ref={parentRef}
        className="max-h-[min(640px,calc(100vh-18rem))] overflow-auto"
        role="region"
        aria-label="Services table"
      >
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
