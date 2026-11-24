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
import {
  parseReplicationFactor,
  ReplicationFactorSelect,
} from "@/components/backup/replication-factor-select";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { BACKUP_SCHEDULE_PRESETS } from "@/lib/backup-schedule-presets";
import { useGetBackupConfigQuery } from "@/lib/api";
import type { ServiceInstance } from "@/lib/types";
import { useEffect, useMemo, useState } from "react";

export type ConfigureInstanceSave = {
  display_name?: string;
  credentials?: ServiceInstance["credentials"];
  backup?: {
    enabled?: boolean;
    policy_override?: {
      schedule?: string;
      replication_factor?: number;
      data_retention_days?: number;
    };
    clear_policy_override?: boolean;
  };
};

export function ConfigureInstanceDialog({
  instance,
  open,
  onOpenChange,
  onSave,
}: {
  instance: ServiceInstance;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (patch: ConfigureInstanceSave) => Promise<void>;
}) {
  const { data: globalBackup } = useGetBackupConfigQuery(undefined, {
    skip: !open,
  });
  const backupMeta = instance.deployment?.backup;
  const backupSupported = backupMeta?.supported ?? false;
  const effective = backupMeta?.policy;

  const [backupOn, setBackupOn] = useState(
    backupMeta?.enabled ?? instance.backup?.enabled ?? true,
  );
  const [customPolicy, setCustomPolicy] = useState(
    effective?.overridden ?? false,
  );
  const [schedule, setSchedule] = useState(
    effective?.schedule ?? globalBackup?.policy.schedule ?? "0 3 * * *",
  );
  const [customCron, setCustomCron] = useState("");
  const [rf, setRf] = useState(
    String(
      effective?.replication_factor ??
        globalBackup?.policy.replication_factor ??
        1,
    ),
  );
  const [retention, setRetention] = useState(
    String(
      effective?.data_retention_days ??
        globalBackup?.policy.data_retention_days ??
        14,
    ),
  );
  const [displayName, setDisplayName] = useState(
    instance.display_name?.trim() || "",
  );

  const scheduleIsPreset = useMemo(
    () => BACKUP_SCHEDULE_PRESETS.some((p) => p.value === schedule),
    [schedule],
  );

  useEffect(() => {
    if (!open) return;
    setDisplayName(instance.display_name?.trim() || "");
    setBackupOn(backupMeta?.enabled ?? instance.backup?.enabled ?? true);
    const pol = backupMeta?.policy;
    const g = globalBackup?.policy;
    const scheduleOverride =
      !!g && !!pol && pol.schedule !== g.schedule;
    const retentionOverride =
      !!g && !!pol && pol.data_retention_days !== g.data_retention_days;
    setCustomPolicy(scheduleOverride || retentionOverride);
    setSchedule(pol?.schedule ?? g?.schedule ?? "0 3 * * *");
    setRf(String(pol?.replication_factor ?? g?.replication_factor ?? 1));
    setRetention(
      String(pol?.data_retention_days ?? g?.data_retention_days ?? 14),
    );
    if (
      pol?.schedule &&
      !BACKUP_SCHEDULE_PRESETS.some((p) => p.value === pol.schedule)
    ) {
      setCustomCron(pol.schedule);
      setSchedule("custom");
    } else {
      setCustomCron("");
    }
  }, [
    open,
    backupMeta,
    instance.backup?.enabled,
    instance.display_name,
    globalBackup?.policy,
  ]);

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const patch: ConfigureInstanceSave = {};

    const trimmedName = displayName.trim();
    const priorName = instance.display_name?.trim() || "";
    if (trimmedName !== priorName) {
      patch.display_name = trimmedName;
    }

    if (backupSupported) {
      patch.backup = { enabled: backupOn };
      const g = globalBackup?.policy;
      const globalRf = g?.replication_factor ?? 1;
      const globalSchedule = g?.schedule ?? "0 3 * * *";
      const globalRetention = g?.data_retention_days ?? 14;
      const destCount = globalBackup?.enabled_destination_count ?? 0;
      const instanceRf = parseReplicationFactor(rf, destCount);
      const resolvedSchedule =
        schedule === "custom" ? customCron.trim() : schedule.trim();
      const instanceRetention = Math.max(1, parseInt(retention, 10) || 14);

      const policyOverride: NonNullable<
        ConfigureInstanceSave["backup"]
      >["policy_override"] = {};
      if (backupOn && instanceRf !== globalRf) {
        policyOverride.replication_factor = instanceRf;
      }
      if (customPolicy) {
        const sched = resolvedSchedule || globalSchedule;
        if (sched !== globalSchedule) {
          policyOverride.schedule = sched;
        }
        if (instanceRetention !== globalRetention) {
          policyOverride.data_retention_days = instanceRetention;
        }
      }
      if (Object.keys(policyOverride).length > 0) {
        patch.backup.policy_override = policyOverride;
      } else if (effective?.overridden) {
        patch.backup.clear_policy_override = true;
      }
    }

    const data = new FormData(e.currentTarget);
    const body: ServiceInstance["credentials"] = {};
    if (
      instance.type === "postgres" ||
      instance.type === "mysql" ||
      instance.type === "mongodb"
    ) {
      const username = String(data.get("username") || "").trim();
      const password = String(data.get("password") || "");
      if (username) body.username = username;
      if (password) body.password = password;
    } else if (instance.type === "redis") {
      const password = String(data.get("password") || "");
      if (password) body.password = password;
    } else if (instance.type === "rustfs") {
      const access_key = String(data.get("access_key") || "").trim();
      const secret_key = String(data.get("secret_key") || "");
      if (access_key) body.access_key = access_key;
      if (secret_key) body.secret_key = secret_key;
    } else if (instance.type === "qdrant") {
      const access_key = String(data.get("access_key") || "");
      if (access_key) body.access_key = access_key;
    } else if (instance.type === "browserless") {
      const access_key = String(data.get("access_key") || "");
      if (access_key) body.access_key = access_key;
    } else if (instance.type === "searxng") {
      const secret_key = String(data.get("secret_key") || "");
      if (secret_key) body.secret_key = secret_key;
    } else if (instance.type === "redpanda") {
      const username = String(data.get("username") || "").trim();
      const password = String(data.get("password") || "");
      if (username) body.username = username;
      if (password) body.password = password;
    }
    if (Object.keys(body).length > 0) {
      patch.credentials = body;
    }

    await onSave(patch);
    onOpenChange(false);
  }

  const globalRf = globalBackup?.policy.replication_factor ?? 1;
  const parsedRf = parseInt(rf, 10) || globalRf;
  const rfOverridesGlobal = parsedRf !== globalRf;
  const globalLine = globalBackup?.policy
    ? `Global default: RF ×${globalBackup.policy.replication_factor}, ${globalBackup.policy.schedule}, ${globalBackup.policy.data_retention_days}d retention`
    : "Global defaults load from DR policy.";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={(e) => void submit(e)} className="contents">
          <DialogHeader>
            <DialogTitle>
              Configure — {instance.display_name || instance.id}
            </DialogTitle>
            <DialogDescription>
              Rename the display label anytime; the endpoint slug{" "}
              <span className="font-mono text-xs">{instance.id}</span> stays
              fixed so client URLs keep working. Credential changes apply on
              next reconcile. Backup policy overrides the DR page defaults for
              this instance only.
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <div className="space-y-3">
              <Field
                label="Display name"
                name="display_name"
                value={displayName}
                onChange={(e) => setDisplayName(e.currentTarget.value)}
                placeholder={instance.type}
                autoComplete="off"
              />
              <Label className="text-eyebrow">Credentials</Label>
              {(instance.type === "postgres" ||
                instance.type === "mysql" ||
                instance.type === "mongodb") && (
                <>
                  <Field
                    label="Username"
                    name="username"
                    defaultValue={instance.credentials.username}
                  />
                  <Field
                    label="Password"
                    name="password"
                    type="password"
                    placeholder="Leave blank to keep current"
                  />
                  <Field
                    label="Bootstrap database"
                    name="database"
                    defaultValue={
                      instance.credentials.database || "default"
                    }
                    placeholder="default (fixed at launch)"
                  />
                  <p className="text-metadata text-muted-foreground">
                    Additional databases are managed in the service details
                    view.
                  </p>
                </>
              )}
              {instance.type === "redis" && (
                <Field
                  label="Password"
                  name="password"
                  type="password"
                  placeholder="Leave blank to keep current"
                />
              )}
              {instance.type === "rustfs" && (
                <>
                  <Field
                    label="Access key"
                    name="access_key"
                    placeholder="Leave blank to keep current (copy from connection)"
                  />
                  <Field
                    label="Secret key"
                    name="secret_key"
                    type="password"
                    placeholder="Leave blank to keep current"
                  />
                </>
              )}
              {instance.type === "qdrant" && (
                <Field
                  label="API key"
                  name="access_key"
                  type="password"
                  placeholder="Leave blank to keep current"
                />
              )}
              {instance.type === "browserless" && (
                <Field
                  label="Access token"
                  name="access_key"
                  type="password"
                  placeholder="Leave blank to keep current"
                />
              )}
              {instance.type === "searxng" && (
                <Field
                  label="Secret key"
                  name="secret_key"
                  type="password"
                  placeholder="Leave blank to keep current"
                />
              )}
              {instance.type === "redpanda" && (
                <>
                  <Field
                    label="SASL username"
                    name="username"
                    defaultValue={instance.credentials.username}
                  />
                  <Field
                    label="SASL password"
                    name="password"
                    type="password"
                    placeholder="Leave blank to keep current"
                  />
                </>
              )}
            </div>

            <div className="space-y-3 border-t border-border pt-3">
              <Label className="text-eyebrow">Disaster recovery</Label>
              {backupSupported ? (
                <>
                  <p className="text-metadata text-muted-foreground">
                    Method{" "}
                    <span className="font-mono">
                      {backupMeta?.method ?? "—"}
                    </span>
                    {backupMeta?.object_prefix ? (
                      <>
                        {" "}
                        ·{" "}
                        <span className="font-mono text-[11px]">
                          {backupMeta.object_prefix}
                        </span>
                      </>
                    ) : null}
                  </p>
                  <label className="flex items-center justify-between gap-3 text-sm">
                    <span>Include in data backups</span>
                    <Switch
                      checked={backupOn}
                      onCheckedChange={setBackupOn}
                      aria-label="Include instance in DR data backups"
                    />
                  </label>

                  <p className="text-metadata text-muted-foreground">
                    {globalLine}
                  </p>

                  {backupOn ? (
                    <div className="space-y-1.5 rounded-lg border border-border bg-muted/20 p-3">
                      <ReplicationFactorSelect
                        id="instance-replication-factor"
                        value={rf}
                        onChange={setRf}
                        maxCopies={
                          globalBackup?.enabled_destination_count ?? 0
                        }
                      />
                      <p className="text-metadata text-muted-foreground">
                        {rfOverridesGlobal
                          ? `This service uses ×${parsedRf} redundancy; global policy is ×${globalRf}.`
                          : `Matches global redundancy (×${globalRf}). Raise copies here for this service only.`}
                      </p>
                    </div>
                  ) : null}

                  <label className="flex items-center justify-between gap-3 text-sm">
                    <span>Custom schedule &amp; retention</span>
                    <Switch
                      checked={customPolicy}
                      onCheckedChange={setCustomPolicy}
                      aria-label="Override global backup schedule and retention"
                    />
                  </label>

                  {customPolicy && (
                    <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-3">
                      <div>
                        <Label htmlFor="inst-schedule">Frequency</Label>
                        <SelectField
                          id="inst-schedule"
                          className="mt-2"
                          value={scheduleIsPreset ? schedule : "custom"}
                          onValueChange={(v) => {
                            if (v === "custom") {
                              setSchedule("custom");
                            } else {
                              setSchedule(v);
                              setCustomCron("");
                            }
                          }}
                          options={[
                            ...BACKUP_SCHEDULE_PRESETS,
                            { value: "custom", label: "Custom cron…" },
                          ]}
                        />
                        {(schedule === "custom" || !scheduleIsPreset) && (
                          <Field
                            className="mt-2"
                            label="Cron expression"
                            value={
                              customCron || (scheduleIsPreset ? "" : schedule)
                            }
                            onChange={(e) => setCustomCron(e.target.value)}
                            placeholder="0 3 * * *"
                          />
                        )}
                      </div>
                      <Field
                        label="Data retention (days)"
                        value={retention}
                        onChange={(e) => setRetention(e.target.value)}
                        inputMode="numeric"
                      />
                    </div>
                  )}
                </>
              ) : (
                <p className="text-metadata text-muted-foreground">
                  This service type is not in DR data backups.
                </p>
              )}
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit">Save</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
