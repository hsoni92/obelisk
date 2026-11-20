"use client";

import { ConnectionCredentialsPanel } from "@/components/services/connection-credentials";
import { CopyButton } from "@/components/services/copy-button";
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
import { SelectField } from "@/components/ui/select";
import type { CreateInstanceFromBackupResult, ServiceInstance } from "@/lib/api";
import { useGetBackupCatalogQuery } from "@/lib/api";
import { relativeTime } from "@/lib/overview-series";
import { toast } from "@/lib/toast";
import {
  AlertTriangle,
  CheckCircle2,
  Database,
  Box,
  HardDrive,
  Layers,
  Plus,
  Radio,
  Search,
  Server,
  Activity,
} from "lucide-react";
import * as React from "react";

type InstanceType = ServiceInstance["type"];

/** Mirrors pkg/obelisk BackupDataServiceTypes — types with DR data backups. */
const BACKUP_RESTORE_TYPES: InstanceType[] = [
  "postgres",
  "mysql",
  "mongodb",
  "rustfs",
  "qdrant",
];

const TYPES: {
  id: InstanceType;
  label: string;
  hint: string;
  icon: typeof Database;
}[] = [
  {
    id: "postgres",
    label: "PostgreSQL",
    hint: "Relational · included in DR backups",
    icon: Database,
  },
  {
    id: "mysql",
    label: "MySQL",
    hint: "Relational · included in DR backups",
    icon: Server,
  },
  {
    id: "mongodb",
    label: "MongoDB",
    hint: "Document store · included in DR backups",
    icon: Database,
  },
  {
    id: "redis",
    label: "Redis",
    hint: "Cache & queues (not in DR data backups)",
    icon: Layers,
  },
  {
    id: "rustfs",
    label: "RustFS",
    hint: "S3-compatible · DR backups (when worker ships)",
    icon: HardDrive,
  },
  {
    id: "qdrant",
    label: "Qdrant",
    hint: "Vector search · DR backups (when worker ships)",
    icon: Box,
  },
  {
    id: "searxng",
    label: "SearXNG",
    hint: "Privacy-respecting metasearch",
    icon: Search,
  },
  {
    id: "redpanda",
    label: "Redpanda",
    hint: "Kafka-compatible streaming (not in DR data backups)",
    icon: Radio,
  },
  {
    id: "unimeter",
    label: "Unimeter",
    hint: "Usage metering & billing events (not in DR data backups)",
    icon: Activity,
  },
];

export type CreateInstanceInput = {
  type: InstanceType;
  name: string;
  fromBackup?: {
    source_type: string;
    source_instance_id: string;
    backup_id?: string;
  };
};

export type CreateResult = {
  instance: ServiceInstance;
  credentials: ServiceInstance["credentials"];
  use?: string;
  from_backup?: CreateInstanceFromBackupResult;
};

export function CreateInstanceDialog({
  open,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (input: CreateInstanceInput) => Promise<CreateResult>;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {/* Content unmounts on close, so this body's state resets each time. */}
        <CreateInstanceBody
          open={open}
          onClose={() => onOpenChange(false)}
          onCreate={onCreate}
        />
      </DialogContent>
    </Dialog>
  );
}

function catalogSourceKey(serviceType: string, instanceId: string) {
  return `${serviceType}/${instanceId}`;
}

function CreateInstanceBody({
  open,
  onClose,
  onCreate,
}: {
  open: boolean;
  onClose: () => void;
  onCreate: (input: CreateInstanceInput) => Promise<CreateResult>;
}) {
  const [type, setType] = React.useState<InstanceType>("postgres");
  const [name, setName] = React.useState("");
  const [bootstrap, setBootstrap] = React.useState(false);
  const [sourceKey, setSourceKey] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [result, setResult] = React.useState<CreateResult | null>(null);

  const { data: catalog } = useGetBackupCatalogQuery(undefined, {
    skip: !open,
    pollingInterval: 30_000,
  });

  const bootstrapEligible = BACKUP_RESTORE_TYPES.includes(type);

  const backupSources = React.useMemo(() => {
    if (!bootstrapEligible || !catalog?.rows) return [];
    return catalog.rows
      .filter(
        (r) =>
          r.service_type === type &&
          r.backup_count > 0 &&
          (r.active_backup_id || r.backup_count > 0),
      )
      .sort((a, b) => {
        const ta = a.active_backup_at ?? a.last_upload_at ?? "";
        const tb = b.active_backup_at ?? b.last_upload_at ?? "";
        return tb.localeCompare(ta);
      });
  }, [bootstrapEligible, catalog?.rows, type]);

  const effectiveSourceKey = React.useMemo(() => {
    if (!bootstrap || backupSources.length === 0) return "";
    if (
      sourceKey &&
      backupSources.some(
        (r) => catalogSourceKey(r.service_type, r.instance_id) === sourceKey,
      )
    ) {
      return sourceKey;
    }
    const first = backupSources[0];
    return catalogSourceKey(first.service_type, first.instance_id);
  }, [backupSources, bootstrap, sourceKey]);

  const selectedSource = backupSources.find(
    (r) =>
      catalogSourceKey(r.service_type, r.instance_id) === effectiveSourceKey,
  );

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (bootstrap && !selectedSource) {
      toast.error("Pick a backup source", "No backup catalog entries match this type.");
      return;
    }
    setSubmitting(true);
    try {
      const displayName = name.trim() || type;
      setResult(
        await onCreate({
          type,
          name: displayName,
          fromBackup:
            bootstrap && selectedSource
              ? {
                  source_type: selectedSource.service_type,
                  source_instance_id: selectedSource.instance_id,
                  backup_id: selectedSource.active_backup_id,
                }
              : undefined,
        }),
      );
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not create instance",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      {result ? (
        <div className="contents">
          <DialogHeader>
            <div
              className={`flex items-center gap-2 ${result.from_backup && !result.from_backup.restore_ok ? "text-warning" : "text-success"}`}
            >
              {result.from_backup && !result.from_backup.restore_ok ? (
                <AlertTriangle className="size-4" />
              ) : (
                <CheckCircle2 className="size-4" />
              )}
              <DialogTitle>
                {result.from_backup
                  ? result.from_backup.restore_ok
                    ? "Instance created from backup"
                    : "Instance created — restore incomplete"
                  : "Instance created"}
              </DialogTitle>
            </div>
            <DialogDescription>
              <span className="font-medium text-foreground">
                {result.instance.display_name || result.instance.id}
              </span>{" "}
              {result.from_backup?.restore_ok
                ? "was provisioned and seeded from the selected backup."
                : result.from_backup
                  ? "was provisioned; data restore did not finish. You can retry from Backups once the pod is running."
                  : "is ready to start. You can copy the unified connection block here or anytime from the service row or details view."}
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-3 rounded-lg bg-muted p-3">
              {result.from_backup ? (
                <p className="text-metadata text-muted-foreground">
                  Source{" "}
                  <span className="font-mono text-foreground">
                    {result.from_backup.source_type}/
                    {result.from_backup.source_instance_id}
                  </span>
                  {result.from_backup.backup_id
                    ? ` · backup ${result.from_backup.backup_id}`
                    : null}
                </p>
              ) : null}
              {result.from_backup?.restore_error ? (
                <p className="text-sm text-warning">{result.from_backup.restore_error}</p>
              ) : null}
              <div className="flex items-center gap-2 text-sm">
                <span className="text-muted-foreground">Endpoint slug</span>
                <span className="min-w-0 flex-1 truncate font-mono text-xs">
                  {result.instance.id}
                </span>
                <CopyButton value={result.instance.id} />
              </div>
              {result.use?.trim() ? (
                <div className="border-t border-border pt-3">
                  <ConnectionCredentialsPanel
                    connection={{
                      id: result.instance.id,
                      type: result.instance.type,
                      use: result.use,
                    }}
                    enabled
                  />
                </div>
              ) : null}
            </div>
          </DialogBody>
          <DialogFooter>
            <Button onClick={onClose}>Done</Button>
          </DialogFooter>
        </div>
      ) : (
        <form onSubmit={(e) => void submit(e)} className="contents">
          <DialogHeader>
            <DialogTitle>New instance</DialogTitle>
            <DialogDescription>
              Pick a service and a display name you can rename later. Obelisk
              assigns a permanent endpoint slug (used in URLs) and credentials
              automatically.
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="instance-type">Service type</Label>
              <SelectField
                id="instance-type"
                value={type}
                onValueChange={(v) => {
                  setType(v as InstanceType);
                  setBootstrap(false);
                  setSourceKey("");
                }}
                options={TYPES.map((t) => ({ value: t.id, label: t.label }))}
              />
              <p className="text-metadata text-muted-foreground">
                {TYPES.find((t) => t.id === type)?.hint}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="instance-name">Display name</Label>
              <Input
                id="instance-name"
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={`e.g. analytics (default: ${type})`}
                autoComplete="off"
              />
            </div>

            {bootstrapEligible ? (
              <div className="space-y-3 rounded-lg border border-border p-3">
                <label className="flex cursor-pointer items-start gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={bootstrap}
                    onChange={(e) => setBootstrap(e.target.checked)}
                    disabled={backupSources.length === 0}
                  />
                  <span>
                    <span className="font-medium">Bootstrap from backup</span>
                    <span className="mt-0.5 block text-metadata text-muted-foreground">
                      {backupSources.length === 0
                        ? "No backups for this service type yet — create empty, or run a backup on an existing instance first."
                        : "Deploy a new instance and restore the active snapshot from another entry of the same type."}
                    </span>
                  </span>
                </label>
                {bootstrap && backupSources.length > 0 ? (
                  <div className="space-y-2">
                    <Label htmlFor="backup-source">Backup source</Label>
                    <SelectField
                      id="backup-source"
                      value={effectiveSourceKey}
                      onValueChange={setSourceKey}
                      options={backupSources.map((r) => {
                        const label = r.display_name?.trim() || r.instance_id;
                        const when = r.active_backup_at ?? r.last_upload_at;
                        const suffix = when ? ` · ${relativeTime(when)}` : "";
                        const orphan = r.orphan && !r.instance_live ? " · orphan" : "";
                        return {
                          value: catalogSourceKey(r.service_type, r.instance_id),
                          label: `${label}${orphan}${suffix}`,
                        };
                      })}
                    />
                  </div>
                ) : null}
              </div>
            ) : null}
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={submitting || (bootstrap && !selectedSource)}
            >
              {submitting
                ? bootstrap
                  ? "Creating & restoring…"
                  : "Creating…"
                : bootstrap
                  ? "Create & restore"
                  : "Create instance"}
            </Button>
          </DialogFooter>
        </form>
      )}
    </>
  );
}

/** Owns the open state so opening the modal never re-renders the page/table. */
export function NewInstanceButton({
  onCreate,
}: {
  onCreate: (input: CreateInstanceInput) => Promise<CreateResult>;
}) {
  const [open, setOpen] = React.useState(false);
  return (
    <>
      <Button className="shrink-0" onClick={() => setOpen(true)}>
        <Plus /> New instance
      </Button>
      <CreateInstanceDialog
        open={open}
        onOpenChange={setOpen}
        onCreate={onCreate}
      />
    </>
  );
}