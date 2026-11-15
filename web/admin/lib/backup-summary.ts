import type { BackupConfigResponse, BackupDestinationView } from "@/lib/types";

export type BackupTone = "success" | "warning" | "live" | "default";

export type BackupSummary = {
  tone: BackupTone;
  /** Short verdict, e.g. "Protected". */
  label: string;
  /** One-line explanation of the verdict. */
  detail: string;
  totalBytes: number;
  totalObjects: number;
  maxBytes: number;
  enabled: BackupDestinationView[];
  healthy: number;
  lastBackupAt?: string;
};

/** Collapses the backup config into a single verdict plus aggregate size. */
export function summarizeBackup(
  config: BackupConfigResponse | undefined,
): BackupSummary | null {
  if (!config) return null;
  const enabled = config.destinations.filter((d) => d.enabled);
  const healthy = enabled.filter((d) => d.ready).length;
  const totalBytes = enabled.reduce((a, d) => a + (d.bytes ?? 0), 0);
  const totalObjects = enabled.reduce((a, d) => a + (d.objects ?? 0), 0);
  const maxBytes = enabled.reduce((a, d) => Math.max(a, d.bytes ?? 0), 0);
  const uploads = enabled
    .map((d) => d.last_upload_at)
    .filter((x): x is string => !!x)
    .sort();
  const lastBackupAt = config.last_run_at ?? uploads[uploads.length - 1];
  const base = { totalBytes, totalObjects, maxBytes, enabled, healthy, lastBackupAt };

  if (enabled.length === 0) {
    return {
      ...base,
      tone: "warning",
      label: "Not configured",
      detail: "No off-site destination — data is not protected.",
    };
  }
  if (healthy === 0) {
    return {
      ...base,
      tone: "live",
      label: "Failing",
      detail: `All ${enabled.length} destination${enabled.length > 1 ? "s are" : " is"} unreachable.`,
    };
  }
  if (healthy < enabled.length) {
    return {
      ...base,
      tone: "warning",
      label: "Partial",
      detail: `${enabled.length - healthy} of ${enabled.length} destinations need attention.`,
    };
  }
  if (!lastBackupAt) {
    return {
      ...base,
      tone: "warning",
      label: "Waiting",
      detail: "Destinations are ready; no backup has run yet.",
    };
  }
  return {
    ...base,
    tone: "success",
    label: "Protected",
    detail: `${healthy} destination${healthy > 1 ? "s" : ""} healthy.`,
  };
}
