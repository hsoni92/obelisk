"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select";
import { formatBytes } from "@/lib/format-bytes";
import type { BackupDestinationView } from "@/lib/types";
import { AlertTriangle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

export type AffectedBackupService = {
  service_type: string;
  instance_id: string;
  display_name?: string;
  backup_count: number;
};

type Props = {
  target: BackupDestinationView | null;
  allDestinations: BackupDestinationView[];
  replicationFactor: number;
  affectedServices: AffectedBackupService[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  busy: boolean;
  onConfirmRemove: (payload: {
    id: string;
    acknowledgePrimaryRemoval: boolean;
    promoteDestinationId?: string;
  }) => Promise<void>;
};

function serviceLabel(s: AffectedBackupService): string {
  if (s.display_name?.trim()) {
    return s.display_name.trim();
  }
  return `${s.service_type}/${s.instance_id}`;
}

export function DeleteDestinationDialog({
  target,
  allDestinations,
  replicationFactor,
  affectedServices,
  open,
  onOpenChange,
  busy,
  onConfirmRemove,
}: Props) {
  const isPrimary = target?.is_primary ?? false;
  const others = useMemo(
    () => allDestinations.filter((d) => d.id !== target?.id),
    [allDestinations, target?.id],
  );
  const soleDestination = allDestinations.length <= 1;

  const [ack, setAck] = useState(false);
  const [promoteId, setPromoteId] = useState("");
  const [confirmText, setConfirmText] = useState("");

  const confirmPhrase = target?.id ?? "";

  useEffect(() => {
    if (!open) {
      setAck(false);
      setPromoteId("");
      setConfirmText("");
      return;
    }
    const firstOther = others[0]?.id ?? "";
    setPromoteId(firstOther);
    setAck(false);
    setConfirmText("");
  }, [open, others]);

  const promoteOptions = useMemo(
    () =>
      others.map((d) => ({
        value: d.id,
        label: `${d.label} (${d.bucket})`,
      })),
    [others],
  );

  if (!target) return null;

  const typedOk = confirmText.trim() === confirmPhrase;
  const affectedCount = affectedServices.length;

  async function confirm() {
    await onConfirmRemove({
      id: target!.id,
      acknowledgePrimaryRemoval: isPrimary,
      promoteDestinationId:
        isPrimary && !soleDestination ? promoteId : undefined,
    });
    onOpenChange(false);
  }

  const canConfirm =
    !busy &&
    typedOk &&
    (isPrimary
      ? ack && (soleDestination || promoteId.length > 0)
      : true);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {isPrimary ? (
              <AlertTriangle className="size-5 text-destructive" aria-hidden />
            ) : null}
            {isPrimary ? "Remove primary destination?" : "Remove destination?"}
          </DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <div className="rounded-lg border border-border bg-muted/30 p-3 space-y-2 text-sm">
            <p className="font-medium">{target.label}</p>
            <p className="text-metadata text-muted-foreground font-mono truncate">
              {target.endpoint}
            </p>
            <div className="flex flex-wrap gap-2">
              {isPrimary ? <Badge variant="live">Primary</Badge> : null}
              <Badge variant="outline">{target.bucket}</Badge>
              <Badge variant="outline">
                {formatBytes(target.bytes)} · {target.objects ?? 0} objects
              </Badge>
            </div>
          </div>

          <div className="rounded-lg border border-destructive/30 bg-destructive-tint/40 p-3 space-y-2 text-sm">
            <p className="font-medium text-foreground">
              {affectedCount === 0
                ? "No services with backup history on this appliance"
                : soleDestination
                  ? `${affectedCount} service${affectedCount === 1 ? "" : "s"} will lose off-site backup`
                  : `${affectedCount} service${affectedCount === 1 ? "" : "s"} may lose copies on this destination`}
            </p>
            {affectedCount > 0 ? (
              <>
                <p className="text-metadata text-muted-foreground">
                  {soleDestination
                    ? "Removing the only destination abandons DR for these instances until a new target is configured."
                    : `Replication is ${replicationFactor}×. Snapshots that never reached other destinations exist only on this endpoint.`}
                </p>
                <ul className="max-h-36 overflow-y-auto space-y-1 text-sm">
                  {affectedServices.map((s) => (
                    <li
                      key={`${s.service_type}/${s.instance_id}`}
                      className="flex items-center justify-between gap-2 rounded-md bg-background/60 px-2 py-1"
                    >
                      <span className="truncate font-medium">
                        {serviceLabel(s)}
                      </span>
                      <span className="text-metadata text-muted-foreground shrink-0">
                        {s.backup_count > 0
                          ? `${s.backup_count} snapshot${s.backup_count === 1 ? "" : "s"}`
                          : "backed up"}
                      </span>
                    </li>
                  ))}
                </ul>
              </>
            ) : (
              <p className="text-metadata text-muted-foreground">
                Policy still applies, but no instance has uploaded data yet.
              </p>
            )}
          </div>

          {isPrimary ? (
            <>
              <div className="space-y-2 text-sm text-muted-foreground">
                <p className="font-medium text-foreground">What you lose on this appliance</p>
                <ul className="list-disc pl-5 space-y-1">
                  <li>
                    Stored API keys and connection test history for this
                    destination id.
                  </li>
                  <li>
                    Future backups will not target this endpoint unless you add
                    it again.
                  </li>
                  {!soleDestination ? (
                    <li>
                      Another destination becomes primary; this endpoint is
                      dropped from replication.
                    </li>
                  ) : (
                    <li>
                      This is your only destination — off-site backup will be
                      disabled until you configure another target.
                    </li>
                  )}
                </ul>
                <p>
                  Objects already in the bucket are{" "}
                  <strong className="text-foreground font-medium">not</strong>{" "}
                  deleted from S3/R2 by Obelisk. You must empty the bucket
                  separately if required.
                </p>
              </div>

              {!soleDestination ? (
                <div className="space-y-2">
                  <Label className="text-eyebrow">New primary destination</Label>
                  <SelectField
                    value={promoteId}
                    onValueChange={setPromoteId}
                    options={promoteOptions}
                    aria-label="Destination to promote to primary"
                  />
                  <p className="text-metadata text-muted-foreground">
                    The selected destination becomes primary first; this row is
                    then removed.
                  </p>
                </div>
              ) : null}

              <label className="flex items-start gap-3 cursor-pointer">
                <Checkbox
                  checked={ack}
                  onCheckedChange={(v) => setAck(v === true)}
                  aria-label="Acknowledge primary destination removal"
                />
                <span className="text-sm leading-snug">
                  I understand this removes off-site backup configuration for{" "}
                  <span className="font-medium text-foreground">{target.label}</span>{" "}
                  and may leave data only in remote buckets Obelisk no longer
                  manages.
                </span>
              </label>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              Removes credentials and stops using{" "}
              <span className="font-medium text-foreground">{target.label}</span>{" "}
              for new backups. Existing objects in{" "}
              <span className="font-mono text-xs">{target.bucket}</span> are not
              deleted from storage.
            </p>
          )}

          <div className="space-y-2">
            <Field
              label="Type destination id to confirm"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={confirmPhrase}
              aria-label="Confirm destination id"
              autoComplete="off"
            />
            <p className="text-metadata text-muted-foreground">
              Enter{" "}
              <code className="font-mono text-[11px] text-foreground">
                {confirmPhrase}
              </code>{" "}
              to enable removal.
            </p>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" type="button" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={!canConfirm}
            onClick={() => void confirm()}
          >
            Remove destination
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}