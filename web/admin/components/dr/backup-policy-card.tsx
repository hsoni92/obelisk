"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { BACKUP_SCHEDULE_PRESETS } from "@/lib/backup-schedule-presets";
import type { BackupConfigResponse } from "@/lib/types";
import { useState } from "react";

const SCHEDULE_PRESETS = [
  ...BACKUP_SCHEDULE_PRESETS,
  { value: "custom", label: "Custom cron…" },
];

type Props = {
  config: BackupConfigResponse;
  busy: boolean;
  onSave: (patch: {
    schedule?: string;
    replication_factor?: number;
    data_retention_days?: number;
    config_keep_count?: number;
    include_config?: boolean;
  }) => Promise<void>;
};

export function BackupPolicyCard({ config, busy, onSave }: Props) {
  const p = config.policy;
  const [schedule, setSchedule] = useState(p.schedule);
  const [rf, setRf] = useState(String(p.replication_factor));
  const [dataDays, setDataDays] = useState(String(p.data_retention_days));
  const [configKeep, setConfigKeep] = useState(String(p.config_keep_count));
  const [includeConfig, setIncludeConfig] = useState(p.include_config);
  const [customCron, setCustomCron] = useState(
    SCHEDULE_PRESETS.some((s) => s.value === p.schedule && s.value !== "custom")
      ? ""
      : p.schedule,
  );
  const scheduleIsCustom =
    schedule === "custom" ||
    !SCHEDULE_PRESETS.some((s) => s.value === schedule && s.value !== "custom");

  async function save() {
    const resolvedSchedule = scheduleIsCustom
      ? (customCron.trim() || p.schedule)
      : schedule.trim() || p.schedule;
    await onSave({
      schedule: resolvedSchedule,
      replication_factor: Math.max(1, parseInt(rf, 10) || 1),
      data_retention_days: Math.max(1, parseInt(dataDays, 10) || 14),
      config_keep_count: Math.max(1, parseInt(configKeep, 10) || 30),
      include_config: includeConfig,
    });
  }

  return (
    <div className="rounded-xl border border-border bg-card p-4 shadow-rb-2 space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-section-title">Global backup policy</h2>
        {config.replication_ready ? (
          <Badge variant="success">RF {p.replication_factor} met</Badge>
        ) : (
          <Badge variant="warning">
            RF {p.replication_factor}: {config.ready_destination_count}/
            {config.enabled_destination_count} ready
          </Badge>
        )}
      </div>
      <p className="text-metadata text-muted-foreground">
        Machine bucket <span className="font-mono">{config.machine_id}</span> —
        data backups for{" "}
        <span className="font-mono">
          {(config.scope?.data_service_types ?? [
            "postgres",
            "mysql",
            "mongodb",
            "rustfs",
            "qdrant",
          ]).join(", ")}
        </span>{" "}
        only (not Redis). Defaults for every instance — override per service
        under Services → Configure.
      </p>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <Label htmlFor="dr-schedule">Schedule</Label>
          <select
            id="dr-schedule"
            className="mt-1 w-full rounded-md border border-border bg-background px-2 py-1.5 text-sm"
            value={schedule}
            onChange={(e) => setSchedule(e.target.value)}
          >
            {SCHEDULE_PRESETS.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
            {scheduleIsCustom && schedule !== "custom" && (
              <option value={schedule}>Current: {schedule}</option>
            )}
          </select>
          {(schedule === "custom" || scheduleIsCustom) && (
            <Field
              className="mt-2"
              label="Cron expression"
              value={customCron || schedule}
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
          value={dataDays}
          onChange={(e) => setDataDays(e.target.value)}
          inputMode="numeric"
        />
        <Field
          label="Config snapshots to keep"
          value={configKeep}
          onChange={(e) => setConfigKeep(e.target.value)}
          inputMode="numeric"
        />
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <label className="flex items-center gap-2 text-sm">
          <Switch
            checked={includeConfig}
            onCheckedChange={setIncludeConfig}
            aria-label="Include control plane in backups"
          />
          Include control-plane config
        </label>
        <Button disabled={busy} onClick={() => void save()}>
          Save policy
        </Button>
      </div>
    </div>
  );
}