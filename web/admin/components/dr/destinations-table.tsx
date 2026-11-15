"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  BACKUP_REMOTE_PROVIDER_OPTIONS,
  parseBackupRemoteEndpoint,
} from "@/lib/backup-remote-endpoint";
import { formatBytes, formatWhen } from "@/lib/format-bytes";
import type { BackupDestinationView } from "@/lib/types";
import { cn } from "@/lib/utils";
import { FlaskConical, Trash2 } from "lucide-react";

function statusVariant(
  status: string,
): "success" | "warning" | "live" | "default" {
  switch (status) {
    case "ready":
      return "success";
    case "disabled":
      return "default";
    case "offline":
      return "live";
    default:
      return "warning";
  }
}

const GRID =
  "grid grid-cols-[minmax(120px,1fr)_minmax(100px,0.7fr)_88px_80px_minmax(88px,0.6fr)_minmax(88px,0.6fr)_auto] items-center gap-2";

type Props = {
  rows: BackupDestinationView[];
  busyKey: string | null;
  onTest: (id: string) => void;
  onToggle: (id: string, enabled: boolean) => void;
  onRemove: (id: string) => void;
};

export function DestinationsTable({
  rows,
  busyKey,
  onTest,
  onToggle,
  onRemove,
}: Props) {
  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-6 text-center">
        No off-site destinations yet. Add S3 or R2 to enable replication.
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <div
        className={cn(GRID, "text-metadata text-muted-foreground px-2 pb-2")}
        role="row"
      >
        <span>Name</span>
        <span>Endpoint</span>
        <span>Status</span>
        <span>Size</span>
        <span>Last test</span>
        <span>Lag</span>
        <span className="sr-only">Actions</span>
      </div>
      <ul className="space-y-1">
        {rows.map((row) => {
          const parsed = parseBackupRemoteEndpoint(row.endpoint);
          const provider =
            row.provider ||
            BACKUP_REMOTE_PROVIDER_OPTIONS.find((o) => o.value === parsed.provider)
              ?.label ||
            parsed.provider;
          return (
            <li
              key={row.id}
              className={cn(
                GRID,
                "rounded-lg border border-border/60 bg-background/50 px-2 py-2 text-sm",
              )}
            >
              <div className="min-w-0">
                <p className="font-medium truncate">{row.label}</p>
                <p className="text-metadata text-muted-foreground truncate">
                  {provider} · {row.bucket}
                </p>
              </div>
              <p
                className="text-metadata text-muted-foreground truncate font-mono"
                title={row.endpoint}
              >
                {row.endpoint.replace(/^https?:\/\//, "")}
              </p>
              <Badge variant={statusVariant(row.status)}>{row.status}</Badge>
              <span title={`${row.objects ?? 0} objects`}>
                {formatBytes(row.bytes)}
              </span>
              <span className="text-metadata">{formatWhen(row.last_test_at)}</span>
              <span className="text-metadata">{row.schedule_lag || "—"}</span>
              <div className="flex justify-end gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!!busyKey}
                  onClick={() => onTest(row.id)}
                  title="Test connection"
                >
                  <FlaskConical className="size-3.5" aria-hidden />
                  <span className="sr-only">Test</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!!busyKey}
                  onClick={() => onToggle(row.id, !row.enabled)}
                >
                  {row.enabled ? "Disable" : "Enable"}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!!busyKey || row.id === "primary"}
                  onClick={() => onRemove(row.id)}
                  title="Remove"
                >
                  <Trash2 className="size-3.5 text-destructive" aria-hidden />
                </Button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}