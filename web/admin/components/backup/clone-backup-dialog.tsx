"use client";

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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { rtkErrorMessage, useCloneBackupMutation } from "@/lib/api";
import { relativeTime } from "@/lib/overview-series";
import { toast } from "@/lib/toast";
import * as React from "react";

export type CloneBackupTarget = {
  sourceType: string;
  sourceInstanceId: string;
  /** Snapshot scope: active when omitted (DR). Lineage row passes node id. */
  backupId?: string;
  backupAt?: string;
  sourceLabel?: string;
};

type Scope = "snapshot" | "lineage";

type Props = {
  target: CloneBackupTarget | null;
  onOpenChange: (open: boolean) => void;
};

export function CloneBackupDialog({ target, onOpenChange }: Props) {
  const [label, setLabel] = React.useState("");
  const [scope, setScope] = React.useState<Scope>("snapshot");
  const [cloneBackup, { isLoading }] = useCloneBackupMutation();

  React.useEffect(() => {
    if (!target) return;
    const base = target.sourceLabel ?? target.sourceInstanceId;
    setLabel(`Clone of ${target.sourceType}/${base}`);
    setScope("snapshot");
  }, [target]);

  async function onSubmit() {
    if (!target) return;
    try {
      const res = await cloneBackup({
        source_type: target.sourceType,
        source_instance_id: target.sourceInstanceId,
        scope,
        backup_id:
          scope === "snapshot" ? target.backupId?.trim() || undefined : undefined,
        label: label.trim() || undefined,
      }).unwrap();
      toast.success(
        "Backup entry cloned",
        `${res.service_type}/${res.instance_id} · ${res.node_count ?? 0} snapshot(s)`,
      );
      onOpenChange(false);
    } catch (e) {
      toast.error("Clone failed", rtkErrorMessage(e));
    }
  }

  return (
    <Dialog open={!!target} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {target ? (
          <>
            <DialogHeader>
              <DialogTitle>Clone backup entry</DialogTitle>
              <DialogDescription>
                Creates a <strong>separate</strong> backup catalog entry (new
                lineage id, orphan). The source service lineage is unchanged.
              </DialogDescription>
            </DialogHeader>
            <DialogBody className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Source{" "}
                <span className="font-mono">
                  {target.sourceType}/{target.sourceInstanceId}
                </span>
                {target.backupAt
                  ? ` · ${relativeTime(target.backupAt)}`
                  : null}
              </p>
              <div className="space-y-2">
                <Label>What to copy</Label>
                <div className="flex flex-col gap-2">
                  <label className="flex cursor-pointer items-start gap-2 text-sm">
                    <input
                      type="radio"
                      name="clone-scope"
                      className="mt-1"
                      checked={scope === "snapshot"}
                      onChange={() => setScope("snapshot")}
                    />
                    <span>
                      <span className="font-medium">Single snapshot</span>
                      <span className="mt-0.5 block text-metadata text-muted-foreground">
                        {target.backupId
                          ? "Copies the snapshot you selected; it becomes active on the new entry."
                          : "Copies the current active snapshot on the source entry."}
                      </span>
                    </span>
                  </label>
                  <label className="flex cursor-pointer items-start gap-2 text-sm">
                    <input
                      type="radio"
                      name="clone-scope"
                      className="mt-1"
                      checked={scope === "lineage"}
                      onChange={() => setScope("lineage")}
                    />
                    <span>
                      <span className="font-medium">Entire lineage</span>
                      <span className="mt-0.5 block text-metadata text-muted-foreground">
                        Duplicates the full backup tree and structure into the
                        new entry.
                      </span>
                    </span>
                  </label>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="clone-entry-label">Entry label</Label>
                <Input
                  id="clone-entry-label"
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  placeholder="Clone of my-db"
                />
              </div>
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={isLoading}
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button
                type="button"
                variant="ink"
                disabled={isLoading}
                onClick={() => void onSubmit()}
              >
                {isLoading ? "Cloning…" : "Clone entry"}
              </Button>
            </DialogFooter>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}