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
  /** Table cells: one truncated line plus copy. */
  compact?: boolean;
  className?: string;
}) {
  const blob = connectionDetailsBlob(connection);
  const lines = connectionCredentialLines(connection);
  const incomplete = connectionCredentialsIncomplete(connection);

  if (!enabled) {
    const text = "Enable this instance to expose remote connection details.";
    return (
      <p
        className={cn(
          "text-metadata text-muted-foreground",
          compact && "truncate",
          className,
        )}
        title={compact ? text : undefined}
      >
        {text}
      </p>
    );
  }

  if (lines.length === 0) {
    const text = "Waiting for connection details…";
    return (
      <p
        className={cn(
          "text-metadata text-muted-foreground",
          compact && "truncate",
          className,
        )}
        title={compact ? text : undefined}
      >
        {text}
      </p>
    );
  }

  if (compact) {
    const shown = lines[0]?.value ?? "";
    const full = blob ?? lines.map((row) => `${row.label}: ${row.value}`).join("\n");
    const title = incomplete
      ? `${full}\n\nSome secrets are missing from the config.`
      : full;
    return (
      <div
        className={cn(
          "flex min-w-0 items-center gap-1.5 overflow-hidden",
          className,
        )}
        title={title}
      >
        <span
          className={cn(
            "min-w-0 flex-1 truncate whitespace-nowrap font-mono text-[11px] leading-none",
            incomplete ? "text-warning" : "text-foreground",
          )}
        >
          {shown}
        </span>
        <CopyButton value={full} label="Copy connection" />
      </div>
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
      <dl className="space-y-1.5">
        {lines.map((row) => (
          <div
            key={`${row.label}-${row.value}`}
            className="grid grid-cols-[minmax(7rem,38%)_1fr_auto] items-start gap-x-2 gap-y-0"
          >
            <dt className="text-eyebrow text-muted-foreground">{row.label}</dt>
            <dd className="min-w-0 break-all font-mono text-xs text-foreground">
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