"use client";

import { BackupRowActionsMenu } from "@/components/backup/backup-row-actions-menu";
import {
  CloneBackupDialog,
  type CloneBackupTarget,
} from "@/components/backup/clone-backup-dialog";
import { ServiceBackupsDialog } from "@/components/services/service-backups-dialog";
import { DRSkeleton } from "@/components/skeletons";
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
import { Input } from "@/components/ui/input";
import {
  rtkErrorMessage,
  useDeleteInstanceBackupForestMutation,
  useGetBackupCatalogQuery,
  useLazyGetStatusQuery,
  useRebuildBackupIndexMutation,
  useRunInstanceBackupMutation,
} from "@/lib/api";
import type { ServiceInstance } from "@/lib/api";
import type { BackupCatalogRow } from "@/lib/types";
import { formatBytes } from "@/lib/format-bytes";
import { relativeTime } from "@/lib/overview-series";
import { serviceRowKey, stateBadgeVariant, type ServiceRow } from "@/lib/service-rows";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { RefreshCw } from "lucide-react";
import * as React from "react";

/** Fixed track widths so header and body rows share one column layout. */
const CATALOG_ROW =
  "grid w-full grid-cols-[minmax(9rem,1.35fr)_5.5rem_4.25rem_3.25rem_minmax(7rem,1fr)_minmax(5.5rem,0.85fr)_2rem] items-center gap-x-3";

export function DRServiceBackupsPanel() {
  const [fetchStatus] = useLazyGetStatusQuery();
  const { data, isLoading, refetch, isFetching } = useGetBackupCatalogQuery(
    undefined,
    {
      pollingInterval: 30_000,
      refetchOnFocus: false,
      refetchOnReconnect: true,
    },
  );
  const [rebuild] = useRebuildBackupIndexMutation();
  const [runBackup] = useRunInstanceBackupMutation();
  const [deleteForest] = useDeleteInstanceBackupForestMutation();
  const [busy, setBusy] = React.useState(false);
  const [rowBusy, setRowBusy] = React.useState<string | null>(null);
  const [viewRow, setViewRow] = React.useState<ServiceRow | null>(null);
  const [cloneTarget, setCloneTarget] =
    React.useState<CloneBackupTarget | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<BackupCatalogRow | null>(
    null,
  );
  const [deleteConfirm, setDeleteConfirm] = React.useState("");

  const rows = data?.rows ?? [];

  const stats = React.useMemo(() => {
    let withSnapshots = 0;
    let orphans = 0;
    let never = 0;
    for (const r of rows) {
      if (r.orphan && !r.instance_live) orphans++;
      if (r.backup_count > 0) withSnapshots++;
      else if (r.instance_live && r.backup_enabled) never++;
    }
    return { withSnapshots, orphans, never, total: rows.length };
  }, [rows]);

  async function syncIndex() {
    setBusy(true);
    try {
      const res = await rebuild().unwrap();
      toast.success(
        "Index refreshed",
        res.nodes_added > 0
          ? `${res.nodes_added} legacy snapshot(s) imported`
          : "Catalog is up to date",
      );
      void refetch();
    } catch (e) {
      toast.error("Refresh failed", rtkErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  async function openLineage(row: BackupCatalogRow) {
    const key = serviceRowKey({ type: row.service_type, id: row.instance_id });
    let inst: ServiceInstance = syntheticInstance(row);
    let state =
      row.orphan && !row.instance_live
        ? "orphan"
        : row.state ?? "unknown";
    try {
      const status = await fetchStatus().unwrap();
      const live = status.instances?.find(
        (i) => i.type === row.service_type && i.id === row.instance_id,
      );
      if (live) {
        inst = live;
      }
      const states = status.serviceStates as Record<string, string> | undefined;
      if (!row.orphan || row.instance_live) {
        state =
          states?.[`${row.service_type}/${row.instance_id}`] ?? state;
      }
    } catch {
      /* catalog row is enough to open read-only lineage */
    }
    setViewRow({
      rowKey: key,
      instance: inst,
      state,
      lastBackupAt: row.last_upload_at,
    });
  }

  async function createBackup(row: BackupCatalogRow) {
    const key = `${row.service_type}/${row.instance_id}`;
    setRowBusy(key);
    try {
      await runBackup({ id: row.instance_id, type: row.service_type }).unwrap();
      toast.success("Backup created", key);
      void refetch();
    } catch (e) {
      toast.error("Backup failed", rtkErrorMessage(e));
    } finally {
      setRowBusy(null);
    }
  }

  async function confirmDeleteForest() {
    if (!deleteTarget) return;
    const phrase = `${deleteTarget.service_type}/${deleteTarget.instance_id}`;
    const want = phrase.toUpperCase();
    if (deleteConfirm.trim().toUpperCase() !== want) {
      toast.warning("Confirmation required", `Type ${want} to confirm.`);
      return;
    }
    setRowBusy(phrase);
    try {
      const res = await deleteForest({
        id: deleteTarget.instance_id,
        type: deleteTarget.service_type,
        confirm: deleteConfirm.trim(),
      }).unwrap();
      toast.success(
        "Backups deleted",
        `${res.deleted?.length ?? 0} snapshot(s) removed`,
      );
      setDeleteTarget(null);
      setDeleteConfirm("");
      void refetch();
    } catch (e) {
      toast.error("Delete failed", rtkErrorMessage(e));
    } finally {
      setRowBusy(null);
    }
  }

  if (isLoading && !data) {
    return <DRSkeleton />;
  }

  return (
    <div className="space-y-4">
      <CloneBackupDialog
        target={cloneTarget}
        onOpenChange={(o) => !o && setCloneTarget(null)}
      />
      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(o) => {
          if (!rowBusy && !o) {
            setDeleteTarget(null);
            setDeleteConfirm("");
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {deleteTarget?.orphan && !deleteTarget?.instance_live
                ? "Delete orphan backups?"
                : "Delete all backups?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              Removes every snapshot and manifest for{" "}
              <span className="font-mono">
                {deleteTarget?.service_type}/{deleteTarget?.instance_id}
              </span>{" "}
              from off-site storage and the local index. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="px-1 pb-2">
            <p className="mb-2 text-metadata text-muted-foreground">
              Type{" "}
              <span className="font-mono uppercase">
                {deleteTarget?.service_type}/{deleteTarget?.instance_id}
              </span>{" "}
              to confirm.
            </p>
            <Input
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              placeholder={`${deleteTarget?.service_type ?? ""}/${deleteTarget?.instance_id ?? ""}`}
              aria-label="Confirm delete all backups"
              autoComplete="off"
            />
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={!!rowBusy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={!!rowBusy}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault();
                void confirmDeleteForest();
              }}
            >
              {rowBusy ? "Deleting…" : "Delete all"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <div className="rounded-xl border border-border bg-card p-4 shadow-rb-2 space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-section-title">Service backups</h2>
            <p className="mt-1 max-w-xl text-sm text-muted-foreground">
              One row per data service. Use the actions menu on each row for
              lineage, backup, or delete (including orphans).
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => void syncIndex()}
          >
            <RefreshCw
              className={cn("mr-1.5 size-4", (isFetching || busy) && "animate-spin")}
              aria-hidden
            />
            Sync from storage
          </Button>
        </div>

        <dl className="grid gap-3 sm:grid-cols-4">
          <Stat label="Services" value={String(stats.total)} />
          <Stat label="With snapshots" value={String(stats.withSnapshots)} />
          <Stat
            label="Never backed up"
            value={String(stats.never)}
            tone={stats.never > 0 ? "warning" : "default"}
          />
          <Stat
            label="Orphan lineages"
            value={String(stats.orphans)}
            tone={stats.orphans > 0 ? "live" : "default"}
          />
        </dl>

        {rows.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            No data services on this appliance yet. Add postgres, mysql, or
            mongodb under Services — they appear here when DR backups apply.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <div className="min-w-[40rem] space-y-1 px-1">
              <div
                className={cn(
                  CATALOG_ROW,
                  "text-metadata text-muted-foreground px-2 pb-1.5",
                )}
                role="row"
              >
                <span>Service</span>
                <span>Status</span>
                <span>In DR</span>
                <span className="tabular-nums">Count</span>
                <span>Active snapshot</span>
                <span>Last upload</span>
                <span className="sr-only">Actions</span>
              </div>
              {rows.map((row) => (
                <CatalogRow
                  key={`${row.service_type}/${row.instance_id}`}
                  row={row}
                  rowBusy={rowBusy === `${row.service_type}/${row.instance_id}`}
                  onView={() => void openLineage(row)}
                  onCreateBackup={() => void createBackup(row)}
                  onDeleteAll={() => {
                    setDeleteTarget(row);
                    setDeleteConfirm("");
                  }}
                  onCloneBackup={() => {
                    setCloneTarget({
                      sourceType: row.service_type,
                      sourceInstanceId: row.instance_id,
                      backupAt: row.active_backup_at,
                      sourceLabel: row.display_name || row.instance_id,
                    });
                  }}
                />
              ))}
            </div>
          </div>
        )}
      </div>

      <ServiceBackupsDialog
        row={viewRow}
        onOpenChange={(o) => !o && setViewRow(null)}
      />
    </div>
  );
}

function Stat({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: string;
  tone?: "default" | "warning" | "live";
}) {
  return (
    <div className="rounded-lg bg-muted/50 px-3 py-2">
      <dt className="text-metadata text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "text-xl font-light tabular-nums",
          tone === "warning" && "text-warning",
          tone === "live" && "text-broadcast",
        )}
      >
        {value}
      </dd>
    </div>
  );
}

function CatalogRow({
  row,
  rowBusy,
  onView,
  onCreateBackup,
  onDeleteAll,
  onCloneBackup,
}: {
  row: BackupCatalogRow;
  rowBusy: boolean;
  onView: () => void;
  onCreateBackup: () => void;
  onDeleteAll: () => void;
  onCloneBackup: () => void;
}) {
  const label = row.display_name || row.instance_id;
  const orphanOnly = row.orphan && !row.instance_live && !row.clone;
  const cloneEntry = row.clone && !row.instance_live;
  const canCreate =
    row.instance_live &&
    row.backup_enabled &&
    row.state === "running";
  const canDeleteAll = row.backup_count > 0;
  const canClone = row.backup_count > 0;
  const menuLabel = `${row.service_type}/${row.instance_id}`;

  return (
    <div
      className={cn(
        CATALOG_ROW,
        "rounded-lg px-2 py-2 text-sm",
        orphanOnly || cloneEntry
          ? "bg-muted/20 opacity-90"
          : "bg-muted/30",
      )}
      role="row"
    >
      <ServiceNameCell
        type={row.service_type}
        label={label}
        id={row.instance_id}
        onOpen={
          row.backup_count > 0 || row.instance_live ? onView : undefined
        }
      />
      {cloneEntry ? (
        <Badge variant="outline" className="w-fit font-normal">
          Clone
        </Badge>
      ) : orphanOnly ? (
        <Badge variant="live" className="w-fit font-normal">
          Orphan
        </Badge>
      ) : row.state ? (
        <Badge variant={stateBadgeVariant(row.state)} className="w-fit font-normal">
          {row.state}
        </Badge>
      ) : (
        <span className="text-muted-foreground">—</span>
      )}
      {!row.instance_live ? (
        <span className="text-muted-foreground">—</span>
      ) : row.backup_enabled ? (
        <Badge variant="success" className="w-fit font-normal">
          On
        </Badge>
      ) : (
        <Badge variant="outline" className="w-fit font-normal">
          Off
        </Badge>
      )}
      <span className="tabular-nums">{row.backup_count}</span>
      <span
        className="min-w-0 truncate text-[13px]"
        title={row.active_backup_at}
      >
        {row.active_backup_at ? (
          <>
            {relativeTime(row.active_backup_at)}
            {row.active_backup_bytes
              ? ` · ${formatBytes(row.active_backup_bytes)}`
              : ""}
          </>
        ) : (
          <span className="text-muted-foreground">None</span>
        )}
      </span>
      <span className="truncate text-[13px] text-muted-foreground">
        {row.last_upload_at ? relativeTime(row.last_upload_at) : "Never"}
      </span>
      <span className="flex justify-end">
        <BackupRowActionsMenu
          label={menuLabel}
          busy={rowBusy}
          orphan={orphanOnly}
          onViewLineage={
            row.backup_count > 0 || row.instance_live ? onView : undefined
          }
          onCreateBackup={canCreate ? onCreateBackup : undefined}
          onCloneBackup={canClone ? onCloneBackup : undefined}
          onDeleteAll={canDeleteAll ? onDeleteAll : undefined}
        />
      </span>
    </div>
  );
}

function ServiceNameCell({
  type,
  label,
  id,
  onOpen,
}: {
  type: string;
  label: string;
  id: string;
  onOpen?: () => void;
}) {
  const body = (
    <>
      <p className="truncate font-medium capitalize">{label}</p>
      <p className="truncate font-mono text-[11px] text-muted-foreground">
        {type}/{id}
      </p>
    </>
  );
  if (!onOpen) {
    return <div className="min-w-0">{body}</div>;
  }
  return (
    <button
      type="button"
      onClick={onOpen}
      title="View backup lineage"
      className="min-w-0 rounded-md text-left outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-ring [&:hover_p:first-child]:underline [&:hover_p:first-child]:underline-offset-2"
    >
      {body}
    </button>
  );
}

function syntheticInstance(row: BackupCatalogRow): ServiceInstance {
  return {
    id: row.instance_id,
    type: row.service_type as ServiceInstance["type"],
    display_name: row.display_name,
    enabled: false,
    credentials: {},
    deployment: {
      release: "",
      namespace: "",
      backup: {
        supported: true,
        enabled: false,
        method: "",
        policy: {
          schedule: "",
          replication_factor: 0,
          data_retention_days: 0,
          overridden: false,
        },
      },
    },
  };
}