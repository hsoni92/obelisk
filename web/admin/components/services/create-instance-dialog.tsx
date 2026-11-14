"use client";

import { CopyButton } from "@/components/services/copy-button";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select";
import type { ServiceInstance } from "@/lib/api";
import {
  CheckCircle2,
  Database,
  Box,
  HardDrive,
  Layers,
  Plus,
  Search,
  Server,
} from "lucide-react";
import * as React from "react";

type InstanceType = ServiceInstance["type"];

const TYPES: {
  id: InstanceType;
  label: string;
  hint: string;
  icon: typeof Database;
}[] = [
  {
    id: "postgres",
    label: "PostgreSQL",
    hint: "Relational database",
    icon: Database,
  },
  { id: "mysql", label: "MySQL", hint: "Relational database", icon: Server },
  { id: "redis", label: "Redis", hint: "Cache & queues", icon: Layers },
  {
    id: "rustfs",
    label: "RustFS",
    hint: "S3-compatible storage",
    icon: HardDrive,
  },
  {
    id: "qdrant",
    label: "Qdrant",
    hint: "Vector search & embeddings",
    icon: Box,
  },
  {
    id: "searxng",
    label: "SearXNG",
    hint: "Privacy-respecting metasearch",
    icon: Search,
  },
];

const CRED_LABELS: Record<string, string> = {
  username: "Username",
  password: "Password",
  database: "Database",
  access_key: "Access key",
  secret_key: "Secret key",
  api_key: "API key",
};

export type CreateResult = {
  instance: ServiceInstance;
  credentials: ServiceInstance["credentials"];
};

export function CreateInstanceDialog({
  open,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (type: InstanceType, name: string) => Promise<CreateResult>;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        {/* Content unmounts on close, so this body's state resets each time. */}
        <CreateInstanceBody
          onClose={() => onOpenChange(false)}
          onCreate={onCreate}
        />
      </DialogContent>
    </Dialog>
  );
}

function CreateInstanceBody({
  onClose,
  onCreate,
}: {
  onClose: () => void;
  onCreate: (type: InstanceType, name: string) => Promise<CreateResult>;
}) {
  const [type, setType] = React.useState<InstanceType>("postgres");
  const [name, setName] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [result, setResult] = React.useState<CreateResult | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      setResult(await onCreate(type, name.trim() || type));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not create instance",
      );
    } finally {
      setSubmitting(false);
    }
  }

  const creds = result
    ? Object.entries({
        ...result.instance.credentials,
        ...result.credentials,
      }).filter(([, v]) => !!v)
    : [];

  return (
    <>
      {result ? (
        <div className="space-y-4">
          <DialogHeader>
            <div className="flex items-center gap-2 text-success">
              <CheckCircle2 className="size-4" />
              <DialogTitle>Instance created</DialogTitle>
            </div>
            <DialogDescription>
              <span className="font-medium text-foreground">
                {result.instance.display_name || result.instance.id}
              </span>{" "}
              is ready to start. Copy these credentials now — secrets are masked
              after you close this window (you can rotate them later).
            </DialogDescription>
          </DialogHeader>
          <dl className="space-y-1.5 rounded-lg bg-muted p-3">
            {creds.map(([k, v]) => (
              <div key={k} className="flex items-center gap-2">
                <dt className="w-24 shrink-0 text-eyebrow text-muted-foreground">
                  {CRED_LABELS[k] ?? k}
                </dt>
                <dd className="min-w-0 flex-1 truncate font-mono text-xs">
                  {v}
                </dd>
                <CopyButton value={String(v)} />
              </div>
            ))}
          </dl>
          <DialogFooter>
            <Button onClick={onClose}>Done</Button>
          </DialogFooter>
        </div>
      ) : (
        <form onSubmit={(e) => void submit(e)} className="space-y-4">
          <DialogHeader>
            <DialogTitle>New instance</DialogTitle>
            <DialogDescription>
              Pick a service and give it a name. We generate secure credentials
              for you.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-1">
            <Label htmlFor="instance-type">Service type</Label>
            <SelectField
              id="instance-type"
              value={type}
              onValueChange={(v) => setType(v as InstanceType)}
              options={TYPES.map((t) => ({ value: t.id, label: t.label }))}
            />
            <p className="text-metadata text-muted-foreground">
              {TYPES.find((t) => t.id === type)?.hint}
            </p>
          </div>

          <div className="space-y-1">
            <Label htmlFor="instance-name">Name</Label>
            <Input
              id="instance-name"
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={`e.g. analytics (default: ${type})`}
              autoComplete="off"
            />
          </div>

          {error && (
            <p
              role="alert"
              className="rounded-md bg-broadcast-tint px-3 py-2 text-xs font-medium text-broadcast"
            >
              {error}
            </p>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Creating…" : "Create instance"}
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
  onCreate: (type: InstanceType, name: string) => Promise<CreateResult>;
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
