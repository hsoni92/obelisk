"use client";

import { Button } from "@/components/ui/button";
import {
  CloneBackupDialog,
  type CloneBackupTarget,
} from "@/components/backup/clone-backup-dialog";
import { BackupLineageSteps } from "@/components/services/backup-lineage-steps";
import type { BackupLineageNode } from "@/lib/types";
import type { ServiceInstance } from "@/lib/api";
import { rtkErrorMessage } from "@/lib/api";
import {
  useDeleteInstanceBackupMutation,
  useGetInstanceBackupLineageQuery,
  useRestoreInstanceActiveBackupMutation,
  useRunInstanceBackupMutation,
} from "@/lib/store/obelisk-api";
import { toast } from "@/lib/toast";
import * as React from "react";

type Props = {
  inst: ServiceInstance;
  enabled: boolean;
  running: boolean;
  compact?: boolean;
  onOpenFull?: () => void;
};

export function BackupLineagePanel({
  inst,
  enabled,
  running,
  compact,
  onOpenFull,
}: Props) {
  const { data, isLoading, refetch } = useGetInstanceBackupLineageQuery(
    { id: inst.id, type: inst.type },
    { skip: !inst.deployment?.backup?.supported },
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

  if (!inst.deployment?.backup?.supported) {
    return null;
  }

  const nodes = data?.nodes ?? [];
  const confirmPhrase = `${inst.type}/${inst.id}`;

  async function onCreateBackup() {
    try {
      const res = await runBackup({ id: inst.id, type: inst.type }).unwrap();
      toast.success("Backup created", res.backups?.[0] ?? "Upload complete");
      void refetch();
    } catch (e) {
      toast.error("Backup failed", rtkErrorMessage(e));
    }
  }

  async function onRestoreActive() {
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

  if (compact) {
    return (
      <div className="space-y-2 py-2">
        <p className="text-metadata text-muted-foreground">
          {nodes.length} backup{nodes.length === 1 ? "" : "s"}
          {data?.active_backup_id ? " · active snapshot set" : ""}
        </p>
        {onOpenFull ? (
          <Button type="button" variant="outline" size="sm" onClick={onOpenFull}>
            Open backup lineage
          </Button>
        ) : null}
      </div>
    );
  }

  function openClone(node: BackupLineageNode) {
    setCloneTarget({
      sourceType: inst.type,
      sourceInstanceId: inst.id,
      backupId: node.id,
      backupAt: node.created_at,
      sourceLabel: inst.display_name || inst.id,
    });
  }

  return (
    <div className="space-y-3 py-1">
      <CloneBackupDialog
        target={cloneTarget}
        onOpenChange={(o) => !o && setCloneTarget(null)}
      />
      <div className="flex flex-wrap gap-1.5">
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
      </div>
      {isLoading ? (
        <p className="text-metadata text-muted-foreground">Loading lineage…</p>
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
    </div>
  );
}