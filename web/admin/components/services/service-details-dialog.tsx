"use client";

import { CopyButton } from "@/components/services/copy-button";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { ServiceInstance } from "@/lib/api";
import { relativeTime } from "@/lib/overview-series";
import {
  connectionEntries,
  stateBadgeVariant,
  type ServiceRow,
} from "@/lib/service-rows";
import * as React from "react";

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h4 className="mb-2 text-eyebrow text-muted-foreground">{title}</h4>
      <div className="rounded-xl bg-muted px-4 py-1">{children}</div>
    </section>
  );
}

function Field({
  label,
  value,
  mono,
  copy,
}: {
  label: string;
  value?: React.ReactNode;
  mono?: boolean;
  copy?: string;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2 text-sm [&:not(:last-child)]:border-b [&:not(:last-child)]:border-border">
      <dt className="shrink-0 text-muted-foreground">{label}</dt>
      <dd className="flex min-w-0 items-center gap-2">
        <span
          className={
            mono ? "truncate font-mono text-xs" : "truncate text-foreground"
          }
          title={typeof value === "string" ? value : undefined}
        >
          {value ?? "—"}
        </span>
        {copy && <CopyButton value={copy} label="Copy" />}
      </dd>
    </div>
  );
}

type Props = {
  row: ServiceRow | null;
  onOpenChange: (open: boolean) => void;
  onConfigure: (inst: ServiceInstance) => void;
};

export function ServiceDetailsDialog({ row, onOpenChange, onConfigure }: Props) {
  const inst = row?.instance;
  const backup = inst?.deployment?.backup;
  const conns = connectionEntries(row?.connection);

  return (
    <Dialog open={!!row} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-lg overflow-y-auto">
        {row && inst && (
          <div className="space-y-5">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-lg font-normal capitalize">
                {inst.display_name || inst.id}
                <Badge variant={stateBadgeVariant(row.state)}>{row.state}</Badge>
              </DialogTitle>
              <DialogDescription className="capitalize">
                {inst.type} instance
              </DialogDescription>
            </DialogHeader>

            <Section title="Identity">
              <dl>
                <Field label="Instance ID" value={inst.id} mono copy={inst.id} />
                <Field label="Type" value={<span className="capitalize">{inst.type}</span>} />
                <Field
                  label="Helm release"
                  value={inst.deployment?.release}
                  mono
                />
                <Field
                  label="Namespace"
                  value={inst.deployment?.namespace}
                  mono
                />
                <Field label="Desired state" value={inst.enabled ? "Enabled" : "Disabled"} />
              </dl>
            </Section>

            <Section title="Backup">
              <dl>
                {!backup?.supported ? (
                  <Field label="Backup" value="Not supported" />
                ) : (
                  <>
                    <Field label="Status" value={backup.enabled ? "Enabled" : "Off"} />
                    <Field
                      label="Last backup"
                      value={
                        row.lastBackupAt
                          ? `${relativeTime(row.lastBackupAt)} · ${new Date(row.lastBackupAt).toLocaleString()}`
                          : "Never"
                      }
                    />
                    <Field label="Method" value={backup.method} />
                    {backup.policy && (
                      <>
                        <Field label="Schedule" value={backup.policy.schedule} mono />
                        <Field
                          label="Replication factor"
                          value={String(backup.policy.replication_factor)}
                        />
                        <Field
                          label="Retention"
                          value={`${backup.policy.data_retention_days} days`}
                        />
                        <Field
                          label="Policy"
                          value={backup.policy.overridden ? "Custom override" : "Default"}
                        />
                      </>
                    )}
                    <Field label="Object prefix" value={backup.object_prefix} mono />
                  </>
                )}
              </dl>
            </Section>

            <Section title="Connection">
              <dl>
                {conns.length === 0 ? (
                  <Field
                    label="Endpoint"
                    value={
                      row.state === "running"
                        ? "Pending connection…"
                        : "Start to expose"
                    }
                  />
                ) : (
                  conns.map((c) => (
                    <Field
                      key={c.label}
                      label={c.label}
                      value={c.value}
                      mono
                      copy={c.value}
                    />
                  ))
                )}
              </dl>
            </Section>

            <div className="flex justify-end">
              <Button
                variant="ink"
                onClick={() => {
                  onOpenChange(false);
                  onConfigure(inst);
                }}
              >
                Configure
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
