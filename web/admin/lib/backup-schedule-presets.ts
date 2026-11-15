export const BACKUP_SCHEDULE_PRESETS = [
  { value: "0 */6 * * *", label: "Every 6 hours" },
  { value: "0 3 * * *", label: "Daily at 03:00 UTC" },
  { value: "0 */12 * * *", label: "Every 12 hours" },
] as const;

export function schedulePresetLabel(cron: string): string {
  const hit = BACKUP_SCHEDULE_PRESETS.find((p) => p.value === cron);
  return hit?.label ?? cron;
}