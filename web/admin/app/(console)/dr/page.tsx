"use client";

import { RestoreWizard } from "@/components/blueprint/restore-wizard";
import { DRPanel } from "@/components/dr/dr-panel";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import {
  rtkErrorMessage,
  statusPollOptions,
  useGetBackupConfigQuery,
  useGetStatusQuery,
  useRunBackupMutation,
} from "@/lib/api";
import { useState } from "react";

export default function DRPage() {
  const [busy, setBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionInfo, setActionInfo] = useState<string | null>(null);

  const { data: status, error: statusError } = useGetStatusQuery(
    undefined,
    statusPollOptions,
  );
  const { data: backupCfg } = useGetBackupConfigQuery();
  const [runBackup] = useRunBackupMutation();

  const backupReady = backupCfg?.backup_ready ?? status?.backup?.ready;
  const error = actionError ?? (statusError ? rtkErrorMessage(statusError) : null);

  async function backupNow() {
    setBusy("backup");
    setActionError(null);
    setActionInfo(null);
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
      if (
        res.data_skipped?.length &&
        (!res.data_objects || res.data_objects.length === 0)
      ) {
        setActionError(parts.join(" · "));
      } else {
        setActionInfo(parts.join(" · "));
      }
    } catch (e) {
      setActionError(rtkErrorMessage(e));
    } finally {
      setBusy(null);
    }
  }

  return (
    <>
      <PageHeader section="dr" />

      {error && (
        <p
          className="text-xs font-medium text-foreground bg-broadcast-tint rounded-full px-3 py-1 mb-3 inline-block"
          role="alert"
        >
          {error}
        </p>
      )}
      {actionInfo && !error && (
        <p className="text-xs text-muted-foreground mb-3">{actionInfo}</p>
      )}

      <div className="space-y-4">
        <RestoreWizard />
        <DRPanel onBusy={setBusy} onError={setActionError} />
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