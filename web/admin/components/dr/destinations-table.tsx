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
import { ArrowUpCircle, FlaskConical, Trash2 } from "lucide-react";

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

/** Shared column template for header + body (subgrid rows). */
const TABLE_GRID =
  "grid grid-cols-[minmax(10rem,1.25fr)_minmax(6rem,0.65fr)_5.75rem_3.5rem_7.25rem_5rem_11.5rem] gap-x-3 min-w-[48rem]";

type Props = {
  rows: BackupDestinationView[];
  busyKey: string | null;
  onTest: (id: string) => void;
  onToggle: (id: string, enabled: boolean) => void;
  onPromote: (id: string) => void;
  onRemove: (row: BackupDestinationView) => void;
};

export function DestinationsTable({
  rows,
  busyKey,
  onTest,
  onToggle,
  onPromote,
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
    <div className="overflow-x-auto -mx-1 px-1">
      <div className={cn(TABLE_GRID, "items-center gap-y-1 px-2")}>
        <span className="text-metadata text-muted-foreground pb-2">Name</span>
        <span className="text-metadata text-muted-foreground pb-2">Endpoint</span>
        <span className="text-metadata text-muted-foreground pb-2">Status</span>
        <span className="text-metadata text-muted-foreground pb-2 text-right">
          Size
        </span>
        <span className="text-metadata text-muted-foreground pb-2">Last test</span>
        <span className="text-metadata text-muted-foreground pb-2">Lag</span>
        <span className="sr-only pb-2">Actions</span>

        <ul className="contents">
          {rows.map((row) => {
            const parsed = parseBackupRemoteEndpoint(row.endpoint);
            const provider =
              row.provider ||
              BACKUP_REMOTE_PROVIDER_OPTIONS.find((o) => o.value === parsed.provider)
                ?.label ||
              parsed.provider;
            const isPrimary = row.is_primary ?? row.id === "primary";
            return (
              <li
                key={row.id}
                className={cn(
                  "col-span-full grid grid-cols-subgrid items-center gap-x-3",
                  "rounded-lg border border-border/60 bg-background/50 px-2 py-2 text-sm",
                )}
              >
                <div className="min-w-0 self-center">
                  <p className="font-medium truncate flex items-center gap-2">
                    {row.label}
                    {isPrimary ? (
                      <Badge variant="outline" className="text-[10px] shrink-0">
                        Primary
                      </Badge>
                    ) : null}
                  </p>
                  <p className="text-metadata text-muted-foreground truncate">
                    {provider} · {row.bucket}
                  </p>
                </div>
                <p
                  className="min-w-0 self-center text-metadata text-muted-foreground truncate font-mono"
                  title={row.endpoint}
                >
                  {row.endpoint.replace(/^https?:\/\//, "")}
                </p>
                <div className="self-center">
                  <Badge variant={statusVariant(row.status)}>{row.status}</Badge>
                </div>
                <span
                  className="self-center text-right tabular-nums"
                  title={`${row.objects ?? 0} objects`}
                >
                  {formatBytes(row.bytes)}
                </span>
                <span className="self-center text-metadata whitespace-nowrap">
                  {formatWhen(row.last_test_at)}
                </span>
                <span className="self-center text-metadata whitespace-nowrap">
                  {row.schedule_lag || "—"}
                </span>
                <div className="flex justify-end gap-1 self-center">
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
                  {!isPrimary ? (
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={!!busyKey}
                      onClick={() => onPromote(row.id)}
                      title="Promote to primary"
                    >
                      <ArrowUpCircle className="size-3.5" aria-hidden />
                      <span className="sr-only">Make primary</span>
                    </Button>
                  ) : null}
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
                    disabled={!!busyKey}
                    onClick={() => onRemove(row)}
                    title="Remove destination"
                  >
                    <Trash2 className="size-3.5 text-destructive" aria-hidden />
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}