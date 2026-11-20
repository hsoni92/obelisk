"use client";

import { StateStatus } from "@/components/ui/status-dot";
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
import type { ServiceInstance } from "@/lib/api";
import { relativeTime } from "@/lib/overview-series";
import { ConnectionCredentialsPanel } from "@/components/services/connection-credentials";
import { BackupLineagePanel } from "@/components/services/backup-lineage-panel";
import { InstanceDatabasesPanel } from "@/components/services/instance-databases-panel";
import { type ServiceRow } from "@/lib/service-rows";
import * as React from "react";

function Section({
  title,
  children,
  plain,
}: {
  title: string;
  children: React.ReactNode;
  /** No tinted container: the content brings its own surfaces. */
  plain?: boolean;
}) {
  return (
    <section>
      <h4 className="mb-2 text-eyebrow text-muted-foreground">{title}</h4>
      {plain ? children : <div className="rounded-xl bg-muted px-4 py-1">{children}</div>}
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
  onOpenBackups?: (rowKey: string) => void;
  inClusterNotes?: string;
};

export function ServiceDetailsDialog({
  row,
  onOpenChange,
  onConfigure,
  onOpenBackups,
  inClusterNotes,
}: Props) {
  const inst = row?.instance;
  const backup = inst?.deployment?.backup;
  return (
    <Dialog open={!!row} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        {row && inst && (
          <div className="contents">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-lg font-normal capitalize">
                {inst.display_name || inst.id}
                <StateStatus state={row.state} className="ml-1 text-sm" />
              </DialogTitle>
              <DialogDescription className="capitalize">
                {inst.type} instance
              </DialogDescription>
            </DialogHeader>
            <DialogBody className="space-y-5">
              <Section title="Identity">
                <dl>
                  <Field
                    label="Display name"
                    value={inst.display_name || "—"}
                  />
                  <Field
                    label="Endpoint slug"
                    value={inst.id}
                    mono
                    copy={inst.id}
                  />
                  {row.connection?.public_path ? (
                    <Field
                      label="Public hostname"
                      value={row.connection.public_path}
                      mono
                      copy={row.connection.public_path}
                    />
                  ) : null}
                  <Field
                    label="Type"
                    value={<span className="capitalize">{inst.type}</span>}
                  />
                  <Field
                    label="Helm release"
                    value={inst.deployment?.release}
                    mono
                  />
                  <Field
                    label="Desired state"
                    value={inst.enabled ? "Enabled" : "Disabled"}
                  />
                </dl>
              </Section>

              <Section title="Backup">
                <dl>
                  {!backup?.supported ? (
                    <Field label="Backup" value="Not supported" />
                  ) : (
                    <>
                      <Field
                        label="Status"
                        value={backup.enabled ? "Enabled" : "Off"}
                      />
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
                          <Field
                            label="Schedule"
                            value={backup.policy.schedule}
                            mono
                          />
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
                            value={
                              backup.policy.overridden
                                ? "Custom override"
                                : "Default"
                            }
                          />
                        </>
                      )}
                      <Field
                        label="Object prefix"
                        value={backup.object_prefix}
                        mono
                      />
                      <div className="py-3">
                        <BackupLineagePanel
                          inst={inst}
                          enabled={backup.enabled}
                          running={row.state === "running"}
                          compact
                          onOpenFull={
                            onOpenBackups
                              ? () => onOpenBackups(row.rowKey)
                              : undefined
                          }
                        />
                      </div>
                    </>
                  )}
                </dl>
              </Section>

              {(inst.type === "postgres" ||
                inst.type === "mysql" ||
                inst.type === "mongodb") && (
                <Section title="Logical databases">
                  <InstanceDatabasesPanel
                    inst={inst}
                    enabled={inst.enabled}
                    running={row.state === "running"}
                  />
                </Section>
              )}

              <Section title="Connection URLs" plain>
                <p className="mb-3 text-metadata text-muted-foreground">
                  Use Remote for clients outside the cluster, or In-cluster for
                  other pods (Kubernetes Service DNS).
                </p>
                <ConnectionCredentialsPanel
                  connection={row.connection}
                  enabled={inst.enabled}
                  inClusterNotes={inClusterNotes}
                />
              </Section>
            </DialogBody>
            <DialogFooter>
              <Button
                variant="ink"
                onClick={() => {
                  onOpenChange(false);
                  onConfigure(inst);
                }}
              >
                Configure
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
