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
import type { ServiceInstance } from "@/lib/api";
import { toast } from "@/lib/toast";
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
];

export type CreateResult = {
  instance: ServiceInstance;
  credentials: ServiceInstance["credentials"];
  use?: string;
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
  const [result, setResult] = React.useState<CreateResult | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      setResult(await onCreate(type, name.trim() || type));
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
            <div className="flex items-center gap-2 text-success">
              <CheckCircle2 className="size-4" />
              <DialogTitle>Instance created</DialogTitle>
            </div>
            <DialogDescription>
              <span className="font-medium text-foreground">
                {result.instance.display_name || result.instance.id}
              </span>{" "}
              is ready to start. You can copy the unified connection block here or
              anytime from the service row or details view.
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-3 rounded-lg bg-muted p-3">
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
          <DialogBody>
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
          </DialogBody>
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
