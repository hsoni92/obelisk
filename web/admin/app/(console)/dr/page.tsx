"use client";

import { RestoreWizard } from "@/components/blueprint/restore-wizard";
import { DRPanel } from "@/components/dr/dr-panel";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import {
  rtkErrorMessage,
  rtkQueryLoadFailed,
  statusPollOptions,
  useGetBackupConfigQuery,
  useGetStatusQuery,
  useRunBackupMutation,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { useEffect, useState } from "react";

export default function DRPage() {
  const [busy, setBusy] = useState<string | null>(null);

  const {
    data: status,
    error: statusError,
    isLoading: statusLoading,
    isFetching: statusFetching,
  } = useGetStatusQuery(undefined, statusPollOptions);
  const { data: backupCfg } = useGetBackupConfigQuery();
  const [runBackup] = useRunBackupMutation();

  const backupReady = backupCfg?.backup_ready ?? status?.backup?.ready;

  useEffect(() => {
    if (
      rtkQueryLoadFailed(statusError, status, statusLoading, statusFetching)
    ) {
      toast.error(
        "Could not load appliance status",
        rtkErrorMessage(statusError),
      );
    }
  }, [statusError, status, statusLoading, statusFetching]);

  async function backupNow() {
    setBusy("backup");
    try {
      const res = await runBackup().unwrap();
      const parts = [res.message ?? "Backup accepted"];
      if (res.object_key) parts.push(res.object_key);
      if (res.data_objects?.length) {
        parts.push(`${res.data_objects.length} data file(s)`);
      }
      if (res.data_skipped?.length) {
        parts.push(`skipped: ${res.data_skipped.join("; ")}`);
      }
      const detail = parts.join(" · ");
      if (
        res.data_skipped?.length &&
        (!res.data_objects || res.data_objects.length === 0)
      ) {
        toast.warning("Backup finished with skips", detail);
      } else {
        toast.success("Backup started", detail);
      }
    } catch (e) {
      toast.error("Backup failed", rtkErrorMessage(e));
    } finally {
      setBusy(null);
    }
  }

  return (
    <>
      <PageHeader section="dr" />

      <div className="space-y-4">
        <RestoreWizard />
        <DRPanel onBusy={setBusy} />
        <div className="flex flex-wrap items-center gap-3">
          <Button
            onClick={() => void backupNow()}
            disabled={!!busy || backupReady === false}
            title={
              backupReady === false
                ? "Add a destination, save keys, and pass connection test first"
                : undefined
            }
          >
            Backup now
          </Button>
          {backupReady === false && (
            <span className="text-metadata text-muted-foreground">
              Use Test on a destination row (or Test all) before backup.
            </span>
          )}
          {status?.reconcileMode === "off" && (
            <span className="text-metadata text-muted-foreground">
              Reconcile off: DB dumps need{" "}
              <code className="font-mono text-[11px]">make dev</code> (k3d) or
              local pg_dump with port-forward.
            </span>
          )}
        </div>
      </div>
    </>
  );
}