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
import { RadioGroup } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import * as RadioGroupPrimitive from "radix-ui/radio-group";
import { SelectField } from "@/components/ui/select";
import type { CreateInstanceFromBackupResult, ServiceInstance } from "@/lib/api";
import { useGetBackupCatalogQuery } from "@/lib/api";
import { relativeTime } from "@/lib/overview-series";
import { toast } from "@/lib/toast";
import {
  AlertTriangle,
  Check,
  CheckCircle2,
  Database,
  Box,
  HardDrive,
  Layers,
  Plus,
  Radio,
  Search,
  Server,
  ShieldCheck,
  X,
  Activity,
  ListOrdered,
  Globe,
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

type Category = "Databases" | "Vector & search" | "Object storage" | "Cache & messaging" | "Metering";

const CATEGORY_ORDER: Category[] = [
  "Databases",
  "Vector & search",
  "Object storage",
  "Cache & messaging",
  "Metering",
];

const TYPES: {
  id: InstanceType;
  label: string;
  desc: string;
  category: Category;
  icon: typeof Database;
}[] = [
  { id: "postgres", label: "PostgreSQL", desc: "Relational database for transactional workloads.", category: "Databases", icon: Database },
  { id: "mysql", label: "MySQL", desc: "Popular relational database, wide ecosystem support.", category: "Databases", icon: Server },
  { id: "mongodb", label: "MongoDB", desc: "Flexible document store for JSON-shaped data.", category: "Databases", icon: Database },
  { id: "qdrant", label: "Qdrant", desc: "Vector database for embeddings and similarity search.", category: "Vector & search", icon: Box },
  { id: "searxng", label: "SearXNG", desc: "Privacy-respecting metasearch engine.", category: "Vector & search", icon: Search },
  { id: "rustfs", label: "RustFS", desc: "S3-compatible object storage.", category: "Object storage", icon: HardDrive },
  { id: "redis", label: "Redis", desc: "In-memory cache, queues and pub/sub.", category: "Cache & messaging", icon: Layers },
  { id: "redpanda", label: "Redpanda", desc: "Kafka-compatible streaming platform.", category: "Cache & messaging", icon: Radio },
  { id: "bunqueue", label: "bunqueue", desc: "Bun job queue · TCP 6789 / HTTP 6790.", category: "Cache & messaging", icon: ListOrdered },
  { id: "browserless", label: "Browserless", desc: "Headless Chromium over HTTP/WebSocket · 1 vCPU / 1 GiB cap.", category: "Vector & search", icon: Globe },
  { id: "unimeter", label: "Unimeter", desc: "Usage metering and billing events.", category: "Metering", icon: Activity },
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
      <DialogContent className="max-h-[min(92vh,880px)] max-w-3xl">
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
  const [query, setQuery] = React.useState("");
  const [bootstrap, setBootstrap] = React.useState(false);
  const [sourceKey, setSourceKey] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [result, setResult] = React.useState<CreateResult | null>(null);

  const { data: catalog } = useGetBackupCatalogQuery(undefined, {
    skip: !open,
    pollingInterval: 30_000,
  });

  const bootstrapEligible = BACKUP_RESTORE_TYPES.includes(type);
  const selectedType = TYPES.find((t) => t.id === type);
  const grouped = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    const match = (t: (typeof TYPES)[number]) =>
      !q ||
      `${t.label} ${t.id} ${t.desc} ${t.category}`.toLowerCase().includes(q);
    return CATEGORY_ORDER.map((category) => ({
      category,
      items: TYPES.filter((t) => t.category === category && match(t)),
    })).filter((g) => g.items.length > 0);
  }, [query]);

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
          <DialogHeader className="pl-8">
            <DialogTitle className="text-lg font-normal">New instance</DialogTitle>
            <DialogDescription>
              Choose a service, give it a name you can change later. Obelisk
              assigns the endpoint slug and credentials automatically.
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-8 px-8 py-6">
            <section aria-labelledby="step-service">
              <StepTitle n={1} id="step-service" title="Choose a service" />
              <div className="mt-4 flex h-10 items-center gap-2 rounded-full bg-muted px-4 text-muted-foreground focus-within:ring-2 focus-within:ring-ring">
                <Search className="size-3.5 shrink-0" aria-hidden />
                <input
                  type="search"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search services, e.g. vector, queue, S3…"
                  aria-label="Search services"
                  className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
                />
                {query ? (
                  <button
                    type="button"
                    aria-label="Clear search"
                    onClick={() => setQuery("")}
                    className="flex size-6 items-center justify-center rounded-full hover:bg-card hover:text-foreground"
                  >
                    <X className="size-3.5" />
                  </button>
                ) : null}
              </div>

              <RadioGroup
                value={type}
                onValueChange={(v) => {
                  setType(v as InstanceType);
                  setBootstrap(false);
                  setSourceKey("");
                }}
                aria-label="Service type"
                className="mt-5 gap-6"
              >
                {grouped.map((g) => (
                  <div key={g.category}>
                    <p className="mb-2.5 flex items-center gap-2 text-eyebrow text-muted-foreground">
                      {g.category}
                      <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] tabular-nums">
                        {g.items.length}
                      </span>
                    </p>
                    <div className="grid gap-3 sm:grid-cols-2">
                      {g.items.map((t) => (
                        <ServiceTile key={t.id} t={t} selected={t.id === type} />
                      ))}
                    </div>
                  </div>
                ))}
                {grouped.length === 0 ? (
                  <div className="flex flex-col items-center gap-2 rounded-2xl bg-muted/60 px-4 py-10 text-center">
                    <Search className="size-5 text-muted-foreground" aria-hidden />
                    <p className="text-sm font-medium">No service matches “{query}”</p>
                    <Button type="button" size="sm" variant="outline" onClick={() => setQuery("")}>
                      Clear search
                    </Button>
                  </div>
                ) : null}
              </RadioGroup>
            </section>

            <section aria-labelledby="step-details">
              <StepTitle n={2} id="step-details" title="Name it" />
              <div className="mt-4 space-y-2">
                <Label htmlFor="instance-name">Display name</Label>
                <Input
                  id="instance-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={`e.g. analytics (default: ${type})`}
                  autoComplete="off"
                />
              </div>

              {bootstrapEligible ? (
                <div className="mt-5 space-y-3 rounded-2xl border border-border p-4">
                  <label className="flex cursor-pointer items-start gap-3 text-sm">
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
            </section>
          </DialogBody>
          <DialogFooter className="justify-between px-8">
            <p className="hidden min-w-0 items-center gap-2 truncate text-sm text-muted-foreground sm:flex">
              {selectedType ? (
                <>
                  <selectedType.icon className="size-4 shrink-0" aria-hidden />
                  <span className="truncate">
                    <span className="font-medium text-foreground">{selectedType.label}</span>
                    {" · "}
                    {BACKUP_RESTORE_TYPES.includes(selectedType.id)
                      ? "included in DR backups"
                      : "not in DR data backups"}
                  </span>
                </>
              ) : null}
            </p>
            <div className="flex items-center gap-2">
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
            </div>
          </DialogFooter>
        </form>
      )}
    </>
  );
}

function StepTitle({ n, id, title }: { n: number; id: string; title: string }) {
  return (
    <h3 id={id} className="flex items-center gap-2.5 text-[15px] font-medium">
      <span className="flex size-6 items-center justify-center rounded-full bg-ink text-xs font-semibold text-ink-foreground">
        {n}
      </span>
      {title}
    </h3>
  );
}

function ServiceTile({ t, selected }: { t: (typeof TYPES)[number]; selected: boolean }) {
  const Icon = t.icon;
  const dr = BACKUP_RESTORE_TYPES.includes(t.id);
  return (
    <RadioGroupPrimitive.Item
      value={t.id}
      className={cn(
        "group relative flex items-start gap-3 rounded-2xl p-4 text-left outline-none ring-1 transition-all",
        "focus-visible:ring-2 focus-visible:ring-ring",
        selected
          ? "bg-muted ring-2 ring-ink"
          : "bg-card ring-border hover:-translate-y-px hover:bg-muted/50 hover:shadow-rb-1",
      )}
    >
      <span
        className={cn(
          "flex size-10 shrink-0 items-center justify-center rounded-xl transition-colors",
          selected ? "bg-ink text-ink-foreground" : "bg-muted text-muted-foreground group-hover:text-foreground",
        )}
      >
        <Icon className="size-5" aria-hidden />
      </span>
      <span className="min-w-0 flex-1 pr-6">
        <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-sm font-medium">{t.label}</span>
          {dr ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-success-tint px-1.5 py-0.5 text-[10px] font-medium text-success">
              <ShieldCheck className="size-3" aria-hidden />
              DR backups
            </span>
          ) : null}
        </span>
        <span className="mt-1 block text-metadata leading-relaxed text-muted-foreground">
          {t.desc}
        </span>
      </span>
      <span
        aria-hidden
        className={cn(
          "absolute right-3 top-3 flex size-5 items-center justify-center rounded-full transition-all",
          selected ? "scale-100 bg-ink text-ink-foreground" : "scale-90 bg-transparent ring-1 ring-border",
        )}
      >
        {selected ? <Check className="size-3" strokeWidth={3} /> : null}
      </span>
    </RadioGroupPrimitive.Item>
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