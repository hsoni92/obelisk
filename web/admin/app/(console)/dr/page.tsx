"use client";

import { BackupRemotePanel } from "@/components/dr/backup-remote-panel";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import {
  rtkErrorMessage,
  statusPollOptions,
  useGetStatusQuery,
  useRunBackupMutation,
} from "@/lib/api";
import { useState } from "react";

export default function DRPage() {
  const [busy, setBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const { data: status, error: statusError } = useGetStatusQuery(
    undefined,
    statusPollOptions,
  );
  const [runBackup] = useRunBackupMutation();

  const error = actionError ?? (statusError ? rtkErrorMessage(statusError) : null);

  async function backupNow() {
    setBusy("backup");
    setActionError(null);
    try {
      await runBackup().unwrap();
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
          className="text-xs font-medium text-broadcast bg-broadcast-tint rounded-full px-3 py-1 mb-3 inline-block"
          role="alert"
        >
          {error}
        </p>
      )}

      <div className="space-y-4">
        <BackupRemotePanel onBusy={setBusy} onError={setActionError} />
        <div className="flex flex-wrap items-center gap-3">
          <Button
            onClick={() => void backupNow()}
            disabled={!!busy || status?.backup?.ready === false}
            title={
              status?.backup?.ready === false
                ? "Configure and test remote backup first"
                : undefined
            }
          >
            Backup now
          </Button>
          {status?.backup?.ready === false && (
            <span className="text-metadata text-muted-foreground">
              Save target and pass connection test to enable backup.
            </span>
          )}
        </div>
      </div>
    </>
  );
}