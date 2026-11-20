"use client";

import { useState } from "react";
import { CopyButton } from "@/components/services/copy-button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { copyToClipboard } from "@/lib/copy-to-clipboard";
import {
  Check,
  Copy,
  Eye,
  EyeOff,
  Globe,
  KeyRound,
  Terminal,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import {
  connectionDetailsBlob,
  connectionCredentialLines,
  connectionCredentialsIncomplete,
  type InstanceConnection,
} from "@/lib/service-rows";
import type { ConnectionAudience } from "@/lib/types";
import type { ConnectionEntry } from "@/lib/service-rows";
import { cn } from "@/lib/utils";

function AudienceToggle({
  audience,
  onChange,
  inClusterAvailable,
}: {
  audience: ConnectionAudience;
  onChange: (a: ConnectionAudience) => void;
  inClusterAvailable: boolean;
}) {
  if (!inClusterAvailable) return null;
  const opts: { id: ConnectionAudience; label: string; hint: string }[] = [
    { id: "edge", label: "Remote", hint: "edge hostnames" },
    { id: "in_cluster", label: "In-cluster", hint: "Kubernetes DNS" },
  ];
  return (
    <div
      role="tablist"
      aria-label="Connection audience"
      className="grid grid-cols-2 gap-1 rounded-full bg-card p-1 shadow-rb-1"
    >
      {opts.map((o) => {
        const on = audience === o.id;
        return (
          <button
            key={o.id}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(o.id)}
            className={cn(
              "flex items-baseline justify-center gap-1.5 rounded-full px-3 py-1.5 text-[13px] outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring",
              on
                ? "bg-ink font-medium text-ink-foreground"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {o.label}
            <span className={cn("text-[11px]", on ? "opacity-70" : "opacity-60")}>
              {o.hint}
            </span>
          </button>
        );
      })}
    </div>
  );
}

type GroupId = "endpoints" | "credentials" | "environment";

const SECRET_RE = /(secret|password|passwd|token|api[-_]?key|access[-_]?key)/i;
const USER_RE = /(^|[_-])(user|username|login)$/i;
const ENDPOINT_RE = /(^[a-z][a-z0-9+.-]*:\/\/)|(^[\w.-]+:\d+$)/i;

function groupOf(e: ConnectionEntry): GroupId {
  if (SECRET_RE.test(e.label) || USER_RE.test(e.label)) return "credentials";
  if (ENDPOINT_RE.test(e.value) || /dns|bootstrap|host/i.test(e.label)) return "endpoints";
  return "environment";
}

const GROUPS: { id: GroupId; title: string; hint: string; icon: LucideIcon }[] = [
  { id: "endpoints", title: "Endpoints", hint: "URLs and hosts", icon: Globe },
  { id: "credentials", title: "Credentials", hint: "Users and secrets", icon: KeyRound },
  { id: "environment", title: "Environment", hint: "Variables", icon: Terminal },
];

function RowCopy({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label={copied ? "Copied" : label}
      title={copied ? "Copied" : label}
      onClick={async () => {
        try {
          await copyToClipboard(value);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1600);
        } catch {
          /* clipboard denied */
        }
      }}
      className="flex size-7 shrink-0 items-center justify-center rounded-full text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
    >
      {copied ? <Check className="size-3.5 text-success" /> : <Copy className="size-3.5" />}
    </button>
  );
}

function EntryRow({ entry, secret }: { entry: ConnectionEntry; secret: boolean }) {
  const [shown, setShown] = useState(false);
  const masked = secret && !shown;
  return (
    <div className="group flex items-center gap-2 rounded-xl px-3 py-2 transition-colors hover:bg-muted/60">
      <div className="min-w-0 flex-1">
        <p className="text-eyebrow text-muted-foreground">{entry.label}</p>
        <p
          className={cn(
            "mt-0.5 font-mono text-xs text-foreground",
            masked ? "tracking-widest" : "break-all",
          )}
        >
          {masked ? "••••••••••••" : entry.value}
        </p>
      </div>
      {secret ? (
        <button
          type="button"
          aria-label={shown ? `Hide ${entry.label}` : `Reveal ${entry.label}`}
          aria-pressed={shown}
          onClick={() => setShown((v) => !v)}
          className="flex size-7 shrink-0 items-center justify-center rounded-full text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          {shown ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
        </button>
      ) : null}
      <RowCopy value={entry.value} label={`Copy ${entry.label}`} />
    </div>
  );
}

export function ConnectionCredentialsPanel({
  connection,
  enabled,
  compact,
  className,
  inClusterNotes,
}: {
  connection?: InstanceConnection;
  enabled?: boolean;
  /** Table cells: one truncated line plus copy. */
  compact?: boolean;
  className?: string;
  inClusterNotes?: string;
}) {
  const [audience, setAudience] = useState<ConnectionAudience>("edge");
  const inClusterAvailable = !!connection?.in_cluster?.host;
  const activeAudience = inClusterAvailable ? audience : "edge";

  const blob = connectionDetailsBlob(connection, activeAudience);
  const lines = connectionCredentialLines(connection, activeAudience);
  const incomplete = connectionCredentialsIncomplete(connection, activeAudience);

  if (!enabled) {
    const text = "Enable this instance to expose connection details.";
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

  const grouped = GROUPS.map((g) => ({
    ...g,
    items: lines.filter((l) => groupOf(l) === g.id),
  })).filter((g) => g.items.length > 0);
  const defaultOpen = grouped
    .filter((g) => g.id !== "environment")
    .map((g) => g.id);

  return (
    <div className={cn("space-y-3", className)}>
      <AudienceToggle
        audience={activeAudience}
        onChange={setAudience}
        inClusterAvailable={inClusterAvailable}
      />
      {activeAudience === "in_cluster" && inClusterNotes ? (
        <p className="text-metadata text-muted-foreground">{inClusterNotes}</p>
      ) : null}
      {incomplete ? (
        <div className="flex items-start gap-2 rounded-xl bg-warning-tint px-3 py-2.5 text-metadata text-warning">
          <TriangleAlert className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <p>
            Some secrets are missing from the config; copy may be incomplete
            until reconcile finishes or you rotate credentials in Configure.
          </p>
        </div>
      ) : null}

      <Accordion
        key={`${activeAudience}-${grouped.map((g) => g.id).join()}`}
        type="multiple"
        defaultValue={defaultOpen}
        className="overflow-hidden rounded-2xl bg-card shadow-rb-1"
      >
        {grouped.map((g) => {
          const Icon = g.icon;
          return (
            <AccordionItem
              key={g.id}
              value={g.id}
              className="border-border/60 last:border-b-0"
            >
              <AccordionTrigger className="px-4 py-3.5 hover:no-underline">
                <span className="flex min-w-0 items-center gap-3">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Icon className="size-3.5" aria-hidden />
                  </span>
                  <span className="min-w-0 text-left">
                    <span className="block text-[13px] font-medium text-foreground">
                      {g.title}
                    </span>
                    <span className="block text-[11px] font-normal text-muted-foreground">
                      {g.hint}
                    </span>
                  </span>
                  <span className="ml-1 rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground tabular-nums">
                    {g.items.length}
                  </span>
                </span>
              </AccordionTrigger>
              <AccordionContent className="px-1.5 pb-2">
                {g.items.map((row) => (
                  <EntryRow
                    key={`${row.label}-${row.value}`}
                    entry={row}
                    secret={g.id === "credentials" && SECRET_RE.test(row.label)}
                  />
                ))}
              </AccordionContent>
            </AccordionItem>
          );
        })}
      </Accordion>

      {blob ? (
        <div className="flex items-center justify-between gap-3 px-1">
          <span className="text-metadata text-muted-foreground">
            Everything above as one block
          </span>
          <CopyButton value={blob} label="Copy all credentials" />
        </div>
      ) : null}
    </div>
  );
}
