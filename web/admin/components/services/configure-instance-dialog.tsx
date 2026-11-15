"use client";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { BACKUP_SCHEDULE_PRESETS } from "@/lib/backup-schedule-presets";
import { useGetBackupConfigQuery } from "@/lib/api";
import type { ServiceInstance } from "@/lib/types";
import { useEffect, useMemo, useState } from "react";

export type ConfigureInstanceSave = {
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
  const { data: globalBackup } = useGetBackupConfigQuery(undefined, { skip: !open });
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
    String(effective?.replication_factor ?? globalBackup?.policy.replication_factor ?? 1),
  );
  const [retention, setRetention] = useState(
    String(effective?.data_retention_days ?? globalBackup?.policy.data_retention_days ?? 14),
  );

  const scheduleIsPreset = useMemo(
    () => BACKUP_SCHEDULE_PRESETS.some((p) => p.value === schedule),
    [schedule],
  );

  useEffect(() => {
    if (!open) return;
    setBackupOn(backupMeta?.enabled ?? instance.backup?.enabled ?? true);
    const pol = backupMeta?.policy;
    setCustomPolicy(pol?.overridden ?? false);
    const g = globalBackup?.policy;
    setSchedule(pol?.schedule ?? g?.schedule ?? "0 3 * * *");
    setRf(String(pol?.replication_factor ?? g?.replication_factor ?? 1));
    setRetention(String(pol?.data_retention_days ?? g?.data_retention_days ?? 14));
    if (pol?.schedule && !BACKUP_SCHEDULE_PRESETS.some((p) => p.value === pol.schedule)) {
      setCustomCron(pol.schedule);
      setSchedule("custom");
    } else {
      setCustomCron("");
    }
  }, [open, backupMeta, instance.backup?.enabled, globalBackup?.policy]);

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const patch: ConfigureInstanceSave = {};

    if (backupSupported) {
      patch.backup = { enabled: backupOn };
      if (customPolicy) {
        const resolvedSchedule =
          schedule === "custom" ? customCron.trim() : schedule.trim();
        patch.backup.policy_override = {
          schedule: resolvedSchedule || globalBackup?.policy.schedule,
          replication_factor: Math.max(1, parseInt(rf, 10) || 1),
          data_retention_days: Math.max(1, parseInt(retention, 10) || 14),
        };
      } else {
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
      const database = String(data.get("database") || "").trim();
      if (username) body.username = username;
      if (password) body.password = password;
      if (database) body.database = database;
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
    } else if (instance.type === "searxng") {
      const secret_key = String(data.get("secret_key") || "");
      if (secret_key) body.secret_key = secret_key;
    }
    if (Object.keys(body).length > 0) {
      patch.credentials = body;
    }

    await onSave(patch);
    onOpenChange(false);
  }

  const globalLine = globalBackup?.policy
    ? `Global default: RF ${globalBackup.policy.replication_factor}, ${globalBackup.policy.schedule}, ${globalBackup.policy.data_retention_days}d retention`
    : "Global defaults load from DR policy.";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md max-h-[90vh] overflow-y-auto">
        <form onSubmit={(e) => void submit(e)} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              Configure — {instance.display_name || instance.id}
            </DialogTitle>
            <DialogDescription>
              Credentials apply on next reconcile. Backup policy overrides the DR
              page defaults for this instance only.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
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
                  label="Database"
                  name="database"
                  defaultValue={instance.credentials.database}
                />
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
                  defaultValue={instance.credentials.access_key}
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
            {instance.type === "searxng" && (
              <Field
                label="Secret key"
                name="secret_key"
                type="password"
                placeholder="Leave blank to keep current"
              />
            )}
          </div>

          <div className="space-y-3 border-t border-border pt-3">
            <Label className="text-eyebrow">Disaster recovery</Label>
            {backupSupported ? (
              <>
                <p className="text-metadata text-muted-foreground">
                  Method{" "}
                  <span className="font-mono">{backupMeta?.method ?? "—"}</span>
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

                <p className="text-metadata text-muted-foreground">{globalLine}</p>

                <label className="flex items-center justify-between gap-3 text-sm">
                  <span>Custom backup policy</span>
                  <Switch
                    checked={customPolicy}
                    onCheckedChange={setCustomPolicy}
                    aria-label="Override global backup policy"
                  />
                </label>

                {customPolicy && (
                  <div className="space-y-3 rounded-lg border border-border bg-muted/30 p-3">
                    <div>
                      <Label htmlFor="inst-schedule">Frequency</Label>
                      <select
                        id="inst-schedule"
                        className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm"
                        value={scheduleIsPreset ? schedule : "custom"}
                        onChange={(e) => {
                          const v = e.target.value;
                          if (v === "custom") {
                            setSchedule("custom");
                          } else {
                            setSchedule(v);
                            setCustomCron("");
                          }
                        }}
                      >
                        {BACKUP_SCHEDULE_PRESETS.map((s) => (
                          <option key={s.value} value={s.value}>
                            {s.label}
                          </option>
                        ))}
                        <option value="custom">Custom cron…</option>
                      </select>
                      {(schedule === "custom" || !scheduleIsPreset) && (
                        <Field
                          className="mt-2"
                          label="Cron expression"
                          value={customCron || (scheduleIsPreset ? "" : schedule)}
                          onChange={(e) => setCustomCron(e.target.value)}
                          placeholder="0 3 * * *"
                        />
                      )}
                    </div>
                    <Field
                      label="Replication (min copies)"
                      value={rf}
                      onChange={(e) => setRf(e.target.value)}
                      inputMode="numeric"
                    />
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