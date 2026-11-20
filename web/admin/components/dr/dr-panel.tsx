"use client";

import { CopyableId } from "@/components/ui/copyable-id";
import { DRSkeleton } from "@/components/skeletons";
import { BackupPolicyCard } from "@/components/dr/backup-policy-card";
import {
  DeleteDestinationDialog,
  type AffectedBackupService,
} from "@/components/dr/delete-destination-dialog";
import { DestinationsTable } from "@/components/dr/destinations-table";
import { R2SetupHint } from "@/components/dr/r2-setup-hint";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  AWS_S3_REGIONS,
  BACKUP_REMOTE_PROVIDER_OPTIONS,
  type BackupRemoteProvider,
  resolveBackupRemoteEndpoint,
} from "@/lib/backup-remote-endpoint";
import {
  rtkErrorMessage,
  useDeleteBackupDestinationMutation,
  useGetBackupConfigQuery,
  usePatchBackupDestinationMutation,
  usePatchBackupPolicyMutation,
  usePostBackupDestinationMutation,
  useGetBackupCatalogQuery,
  usePromoteBackupDestinationMutation,
  useTestAllBackupDestinationsMutation,
  useTestBackupDestinationMutation,
} from "@/lib/api";
import { toast } from "@/lib/toast";
import { CardHeading } from "@/components/ui/card";
import type { BackupDestinationView } from "@/lib/types";
import { Cloud, Plus } from "lucide-react";
import { useMemo, useState } from "react";

export function DRPanel({ onBusy }: { onBusy: (key: string | null) => void }) {
  const { data: config, isLoading } = useGetBackupConfigQuery(undefined, {
    pollingInterval: 8000,
  });
  const { data: catalog } = useGetBackupCatalogQuery(undefined, {
    pollingInterval: 15000,
  });

  const affectedBackupServices = useMemo((): AffectedBackupService[] => {
    const rows = catalog?.rows ?? [];
    return rows
      .filter(
        (r) =>
          r.backup_enabled &&
          (r.backup_count > 0 || Boolean(r.last_upload_at?.trim())),
      )
      .map((r) => ({
        service_type: r.service_type,
        instance_id: r.instance_id,
        display_name: r.display_name,
        backup_count: r.backup_count,
      }));
  }, [catalog?.rows]);
  const [patchPolicy] = usePatchBackupPolicyMutation();
  const [postDestination] = usePostBackupDestinationMutation();
  const [patchDestination] = usePatchBackupDestinationMutation();
  const [deleteDestination] = useDeleteBackupDestinationMutation();
  const [promoteDestination] = usePromoteBackupDestinationMutation();
  const [testDestination] = useTestBackupDestinationMutation();
  const [testAll] = useTestAllBackupDestinationsMutation();

  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [addOpen, setAddOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<BackupDestinationView | null>(
    null,
  );

  const [label, setLabel] = useState("");
  const [provider, setProvider] = useState<BackupRemoteProvider>("r2");
  const [awsRegion, setAwsRegion] = useState("us-east-1");
  const [customEndpoint, setCustomEndpoint] = useState("");
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");

  const endpoint = resolveBackupRemoteEndpoint(
    provider,
    awsRegion,
    customEndpoint,
  );

  async function act(key: string, fn: () => Promise<void>) {
    setBusyKey(key);
    onBusy(key);
    try {
      await fn();
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      setBusyKey(null);
      onBusy(null);
    }
  }

  const statusBadge = useMemo(() => {
    if (!config) return null;
    if (config.backup_ready && config.replication_ready) {
      return <Badge variant="success">ready</Badge>;
    }
    if ((config.destinations ?? []).length === 0) {
      return <Badge variant="warning">add destination</Badge>;
    }
    return <Badge variant="warning">setup</Badge>;
  }, [config]);

  if (!config) {
    return isLoading ? (
      <DRSkeleton />
    ) : (
      <p className="text-sm text-muted-foreground">
        Backup configuration is unavailable.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      <BackupPolicyCard
        config={config}
        busy={!!busyKey}
        onSave={async (patch) => {
          await act("policy", async () => {
            await patchPolicy(patch).unwrap();
            toast.success("Backup policy saved");
          });
        }}
      />

      <div className="overflow-hidden rounded-xl border border-border bg-card p-4 shadow-rb-2 space-y-4">
        <CardHeading
          icon={Cloud}
          tone="blue"
          title="Off-site destinations"
          badge={statusBadge}
          className="-mx-4 -mt-4"
          actions={
            <>
            <Button
              variant="outline"
              size="sm"
              disabled={!!busyKey || (config.destinations ?? []).length === 0}
              onClick={() =>
                act("test-all", async () => {
                  await testAll().unwrap();
                  toast.success("Connection tests finished");
                })
              }
            >
              Test all
            </Button>
            <Button
              size="sm"
              disabled={!!busyKey}
              onClick={() => {
                setLabel("");
                setAccessKey("");
                setSecretKey("");
                setAddOpen(true);
              }}
            >
              <Plus className="size-4 mr-1" aria-hidden />
              Add destination
            </Button>
            </>
          }
        />
        <DestinationsTable
          rows={config.destinations ?? []}
          busyKey={busyKey}
          onTest={(id) =>
            act(`test-${id}`, async () => {
              await testDestination({ id }).unwrap();
              toast.success("Destination test complete");
            })
          }
          onToggle={(id, enabled) =>
            act(`toggle-${id}`, async () => {
              await patchDestination({ id, body: { enabled } }).unwrap();
            })
          }
          onPromote={(id) =>
            act(`promote-${id}`, async () => {
              await promoteDestination({ id }).unwrap();
              toast.success("Primary destination updated");
            })
          }
          onRemove={(row) => setRemoveTarget(row)}
        />
      </div>

      <DeleteDestinationDialog
        target={removeTarget}
        allDestinations={config.destinations ?? []}
        replicationFactor={config.policy.replication_factor}
        affectedServices={affectedBackupServices}
        open={!!removeTarget}
        onOpenChange={(open) => {
          if (!open) setRemoveTarget(null);
        }}
        busy={!!busyKey}
        onConfirmRemove={async (payload) => {
          await act(`del-${payload.id}`, async () => {
            await deleteDestination({
              id: payload.id,
              acknowledge_primary_removal: payload.acknowledgePrimaryRemoval,
              promote_destination_id: payload.promoteDestinationId,
            }).unwrap();
            toast.success("Destination removed");
          });
        }}
      />

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Add S3 / R2 destination</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-3">
              <Field
                label="Label"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="e.g. Cloudflare R2"
              />
              <div className="space-y-2">
                <Label className="text-eyebrow">Provider</Label>
                <SelectField
                  value={provider}
                  onValueChange={(v) => setProvider(v as BackupRemoteProvider)}
                  options={BACKUP_REMOTE_PROVIDER_OPTIONS}
                  aria-label="Remote storage provider"
                />
              </div>
              {provider === "aws" && (
                <div className="space-y-2">
                  <Label className="text-eyebrow">AWS region</Label>
                  <SelectField
                    value={awsRegion}
                    onValueChange={setAwsRegion}
                    options={AWS_S3_REGIONS}
                    aria-label="AWS region"
                  />
                </div>
              )}
              {(provider === "r2" || provider === "custom") && (
                <Field
                  label={provider === "r2" ? "R2 endpoint URL" : "Endpoint URL"}
                  value={customEndpoint}
                  onChange={(e) => setCustomEndpoint(e.target.value)}
                  placeholder="https://<account>.r2.cloudflarestorage.com"
                />
              )}
              {provider === "r2" && <R2SetupHint />}
              <Field
                label="Access key"
                value={accessKey}
                onChange={(e) => setAccessKey(e.target.value)}
                autoComplete="off"
              />
              <Field
                label="Secret key"
                type="password"
                value={secretKey}
                onChange={(e) => setSecretKey(e.target.value)}
                autoComplete="new-password"
              />
              <p className="text-metadata text-muted-foreground">
                Bucket name is{" "}
                <CopyableId value={config.machine_id} label="machine ID" className="align-middle" />{" "}
                (machine ID). Keys are stored in{" "}
                <span className="font-mono">{config.credentials_secret}</span>{" "}
                on disk beside the config in dev.
              </p>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAddOpen(false)}>
              Cancel
            </Button>
            <Button
              disabled={!endpoint || !accessKey || !secretKey}
              onClick={() =>
                act("add-dest", async () => {
                  await postDestination({
                    label: label.trim() || "destination",
                    provider,
                    endpoint,
                    credentials: {
                      access_key_id: accessKey,
                      secret_access_key: secretKey,
                    },
                  }).unwrap();
                  setAddOpen(false);
                  toast.info("Destination added", "Run test to verify upload.");
                })
              }
            >
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
