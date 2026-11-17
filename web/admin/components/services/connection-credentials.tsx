"use client";

import { CopyButton } from "@/components/services/copy-button";
import {
  connectionDetailsBlob,
  connectionCredentialLines,
  connectionCredentialsIncomplete,
  type InstanceConnection,
} from "@/lib/service-rows";
import { cn } from "@/lib/utils";

export function ConnectionCredentialsPanel({
  connection,
  enabled,
  compact,
  className,
}: {
  connection?: InstanceConnection;
  enabled?: boolean;
  /** Table cells: tighter typography, fewer lines before clamp. */
  compact?: boolean;
  className?: string;
}) {
  const blob = connectionDetailsBlob(connection);
  const lines = connectionCredentialLines(connection);
  const incomplete = connectionCredentialsIncomplete(connection);

  if (!enabled) {
    return (
      <p className={cn("text-metadata text-muted-foreground", className)}>
        Enable this instance to expose remote connection details.
      </p>
    );
  }

  if (lines.length === 0) {
    return (
      <p className={cn("text-metadata text-muted-foreground", className)}>
        Waiting for connection details…
      </p>
    );
  }

  return (
    <div className={cn("space-y-2", className)}>
      {incomplete ? (
        <p className="text-metadata text-warning">
          Some secrets are missing from the config; copy may be incomplete until
          reconcile finishes or you rotate credentials in Configure.
        </p>
      ) : null}
      <dl
        className={cn(
          "space-y-1.5",
          compact && "max-h-[6.5rem] overflow-y-auto pr-1",
        )}
      >
        {lines.map((row) => (
          <div
            key={`${row.label}-${row.value}`}
            className="grid grid-cols-[minmax(7rem,38%)_1fr_auto] items-start gap-x-2 gap-y-0"
          >
            <dt className="text-eyebrow text-muted-foreground">{row.label}</dt>
            <dd
              className={cn(
                "min-w-0 break-all font-mono text-foreground",
                compact ? "text-[10px] leading-snug" : "text-xs",
              )}
            >
              {row.value}
            </dd>
            <dd>
              <CopyButton value={row.value} label={`Copy ${row.label}`} />
            </dd>
          </div>
        ))}
      </dl>
      {blob ? (
        <div className="flex items-center gap-2 border-t border-border pt-2">
          <span className="text-metadata text-muted-foreground">
            Full block
          </span>
          <CopyButton value={blob} label="Copy all credentials" />
        </div>
      ) : null}
    </div>
  );
}