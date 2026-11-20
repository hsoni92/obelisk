"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  CloneBackupDialog,
  type CloneBackupTarget,
} from "@/components/backup/clone-backup-dialog";
import { BackupLineageSteps } from "@/components/services/backup-lineage-steps";
import type { BackupLineageNode } from "@/lib/types";
import type { ServiceInstance } from "@/lib/api";
import { rtkErrorMessage } from "@/lib/api";
import { stateBadgeVariant, type ServiceRow } from "@/lib/service-rows";
import {
  useDeleteInstanceBackupMutation,
  useGetInstanceBackupLineageQuery,
  useRestoreInstanceActiveBackupMutation,
  useRunInstanceBackupMutation,
} from "@/lib/store/obelisk-api";
import { toast } from "@/lib/toast";
import { relativeTime } from "@/lib/overview-series";
import * as React from "react";

type Props = {
  row: ServiceRow | null;
  onOpenChange: (open: boolean) => void;
};

export function ServiceBackupsDialog({ row, onOpenChange }: Props) {
  const inst = row?.instance;
  const backup = inst?.deployment?.backup;
  const supported = backup?.supported ?? false;

  const { data, isLoading, refetch, isFetching } =
    useGetInstanceBackupLineageQuery(
      { id: inst?.id ?? "", type: inst?.type ?? "postgres" },
      { skip: !row || !inst || !supported },
    );

  const [runBackup, { isLoading: runBusy }] = useRunInstanceBackupMutation();
  const [restoreActive, { isLoading: restoreBusy }] =
    useRestoreInstanceActiveBackupMutation();
  const [deleteBackup, { isLoading: deleteBusy }] =
    useDeleteInstanceBackupMutation();
  const [confirmDelete, setConfirmDelete] = React.useState<string | null>(null);
  const [confirmText, setConfirmText] = React.useState("");
  const [cloneTarget, setCloneTarget] =
    React.useState<CloneBackupTarget | null>(null);

  function openClone(node: BackupLineageNode) {
    if (!inst) return;
    setCloneTarget({
      sourceType: inst.type,
      sourceInstanceId: inst.id,
      backupId: node.id,
      backupAt: node.created_at,
      sourceLabel: inst.display_name || inst.id,
    });
  }

  const nodes = data?.nodes ?? [];
  const confirmPhrase = inst ? `${inst.type}/${inst.id}` : "";

  async function onCreateBackup() {
    if (!inst) return;
    try {
      const res = await runBackup({ id: inst.id, type: inst.type }).unwrap();
      toast.success("Backup created", res.backups?.[0] ?? "Upload complete");
      void refetch();
    } catch (e) {
      toast.error("Backup failed", rtkErrorMessage(e));
    }
  }

  async function onRestoreActive() {
    if (!inst) return;
    if (
      !window.confirm(
        `Restore active backup on ${inst.type}/${inst.id}? This replaces database data.`,
      )
    ) {
      return;
    }
    try {
      const res = await restoreActive({ id: inst.id, type: inst.type }).unwrap();
      toast.success("Restore complete", res.backup_id ?? "");
    } catch (e) {
      toast.error("Restore failed", rtkErrorMessage(e));
    }
  }

  async function onDeleteNode(backupId: string) {
    if (!inst) return;
    const want = confirmPhrase.toUpperCase();
    if (confirmText.trim().toUpperCase() !== want) {
      toast.warning("Confirmation required", `Type ${want} to delete.`);
      return;
    }
    try {
      await deleteBackup({
        id: inst.id,
        type: inst.type,
        backupId,
        confirm: confirmText.trim(),
      }).unwrap();
      toast.success("Backup deleted", "Subtree removed from storage.");
      setConfirmDelete(null);
      setConfirmText("");
      void refetch();
    } catch (e) {
      toast.error("Delete failed", rtkErrorMessage(e));
    }
  }

  const running = row?.state === "running";
  const enabled = backup?.enabled ?? false;

  return (
    <>
    <CloneBackupDialog
      target={cloneTarget}
      onOpenChange={(o) => !o && setCloneTarget(null)}
    />
    <Dialog open={!!row} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] max-w-3xl flex-col">
        {row && inst && (
          <>
            <DialogHeader>
              <DialogTitle className="flex flex-wrap items-center gap-2 text-lg font-normal capitalize">
                Backups · {inst.display_name || inst.id}
                <Badge variant={stateBadgeVariant(row.state)}>
                  {row.state}
                </Badge>
              </DialogTitle>
              <DialogDescription>
                Per-service lineage — {inst.type}/{inst.id}
                {data?.lineage_id ? (
                  <>
                    {" "}
                    · forest{" "}
                    <span className="font-mono text-xs">
                      {data.lineage_id.slice(0, 8)}
                    </span>
                  </>
                ) : null}
              </DialogDescription>
            </DialogHeader>
            <DialogBody className="min-h-0 flex-1 space-y-5 overflow-y-auto">
              {!supported ? (
                <p className="text-sm text-muted-foreground">
                  This service type is not included in DR data backups.
                </p>
              ) : (
                <>
                  <div className="flex flex-wrap items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={!enabled || !running || runBusy}
                      onClick={() => void onCreateBackup()}
                    >
                      Create backup
                    </Button>
                    <Button
                      type="button"
                      variant="ink"
                      size="sm"
                      disabled={
                        !inst.deployment?.backup?.supported ||
                        !data?.active_backup_id ||
                        restoreBusy ||
                        !running
                      }
                      onClick={() => void onRestoreActive()}
                    >
                      Restore active
                    </Button>
                    {isFetching ? (
                      <span className="text-metadata text-muted-foreground">
                        Refreshing…
                      </span>
                    ) : null}
                  </div>
                  {!running ? (
                    <p className="text-metadata text-muted-foreground">
                      Start the service to create or restore backups.
                    </p>
                  ) : null}
                  {data?.active_backup_id && row.lastBackupAt ? (
                    <p className="text-sm text-muted-foreground">
                      Active snapshot · last upload{" "}
                      {relativeTime(row.lastBackupAt)}
                    </p>
                  ) : null}
                  {isLoading ? (
                    <p className="text-metadata text-muted-foreground">
                      Loading lineage…
                    </p>
                  ) : (
                    <>
                      <BackupLineageSteps
                        nodes={nodes}
                        orphan={data?.orphan}
                        clone={data?.clone}
                        confirmDelete={confirmDelete}
                        confirmText={confirmText}
                        deleteBusy={deleteBusy}
                        onConfirmTextChange={setConfirmText}
                        onStartDelete={(id) => {
                          setConfirmDelete(id);
                          setConfirmText("");
                        }}
                        onCancelDelete={() => {
                          setConfirmDelete(null);
                          setConfirmText("");
                        }}
                        onDelete={(id) => void onDeleteNode(id)}
                        confirmPhrase={confirmPhrase}
                        rowLabel={`${inst.type}/${inst.id}`}
                        onCloneNode={openClone}
                      />
                    </>
                  )}
                </>
              )}
            </DialogBody>
          </>
        )}
      </DialogContent>
    </Dialog>
    </>
  );
}