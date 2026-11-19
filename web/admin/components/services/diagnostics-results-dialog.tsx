"use client";

import { Badge } from "@/components/ui/badge";
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
import type { DiagnosticInstanceResult } from "@/lib/types";
import { cn } from "@/lib/utils";
import { Activity, CheckCircle2, CircleDashed, XCircle } from "lucide-react";

function resultVariant(r: DiagnosticInstanceResult) {
  if (r.skipped) return "muted" as const;
  if (r.ok) return "success" as const;
  return "danger" as const;
}

function ResultIcon({ r }: { r: DiagnosticInstanceResult }) {
  if (r.skipped) {
    return <CircleDashed className="size-4 text-muted-foreground" aria-hidden />;
  }
  if (r.ok) {
    return <CheckCircle2 className="size-4 text-[var(--chart-1)]" aria-hidden />;
  }
  return <XCircle className="size-4 text-destructive" aria-hidden />;
}

export function DiagnosticsResultsDialog({
  open,
  onOpenChange,
  results,
  running,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  results: DiagnosticInstanceResult[] | null;
  running?: boolean;
}) {
  const ok = results?.filter((r) => r.ok && !r.skipped).length ?? 0;
  const failed = results?.filter((r) => !r.ok && !r.skipped).length ?? 0;
  const skipped = results?.filter((r) => r.skipped).length ?? 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-2xl gap-0">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Activity className="size-5" aria-hidden />
            Connection diagnostics
          </DialogTitle>
          <DialogDescription>
            End-to-end probes use the same URLs and credentials shown in the
            Services table (public edge path when on VPS).
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="max-h-[min(60vh,32rem)] overflow-y-auto py-2">
          {running && (!results || results.length === 0) ? (
            <p className="text-sm text-muted-foreground">Running probes…</p>
          ) : null}
          {results && results.length > 0 ? (
            <ul className="space-y-3">
              {results.map((r) => {
                const v = resultVariant(r);
                const title = r.display_name?.trim()
                  ? `${r.display_name} (${r.type}/${r.id})`
                  : `${r.type}/${r.id}`;
                return (
                  <li
                    key={`${r.type}/${r.id}`}
                    className="rounded-xl border border-border bg-muted/40 px-4 py-3"
                  >
                    <div className="flex flex-wrap items-start gap-2">
                      <ResultIcon r={r} />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-sm font-medium text-foreground">
                            {title}
                          </span>
                          <Badge
                            variant={
                              v === "success"
                                ? "success"
                                : v === "danger"
                                  ? "warning"
                                  : "outline"
                            }
                          >
                            {r.skipped
                              ? "Skipped"
                              : r.ok
                                ? "OK"
                                : "Failed"}
                          </Badge>
                          <span className="text-metadata text-muted-foreground">
                            cluster: {r.cluster_state}
                          </span>
                          {r.duration_ms > 0 ? (
                            <span className="text-metadata text-muted-foreground">
                              {r.duration_ms} ms
                            </span>
                          ) : null}
                        </div>
                        <p
                          className={cn(
                            "mt-1 text-sm",
                            r.ok && !r.skipped
                              ? "text-muted-foreground"
                              : r.skipped
                                ? "text-muted-foreground"
                                : "text-foreground",
                          )}
                        >
                          {r.skipped ? r.skipped_reason ?? r.summary : r.summary}
                        </p>
                        {r.public_host ? (
                          <p className="mt-1 truncate font-mono text-[11px] text-muted-foreground">
                            {r.public_host}
                          </p>
                        ) : null}
                        {r.checks && r.checks.length > 0 ? (
                          <ul className="mt-2 space-y-0.5 border-t border-border pt-2">
                            {r.checks.map((c) => (
                              <li
                                key={c.name}
                                className="flex gap-2 font-mono text-[11px]"
                              >
                                <span
                                  className={
                                    c.ok
                                      ? "text-[var(--chart-1)]"
                                      : "text-destructive"
                                  }
                                >
                                  {c.ok ? "✓" : "✗"}
                                </span>
                                <span className="text-muted-foreground">
                                  {c.name}
                                  {c.detail ? `: ${c.detail}` : ""}
                                </span>
                              </li>
                            ))}
                          </ul>
                        ) : null}
                      </div>
                    </div>
                  </li>
                );
              })}
            </ul>
          ) : !running ? (
            <p className="text-sm text-muted-foreground">No results.</p>
          ) : null}
        </DialogBody>
        {results && results.length > 0 ? (
          <DialogFooter className="flex-wrap gap-2 border-t border-border pt-4">
            <p className="mr-auto text-metadata text-muted-foreground">
              {ok} passed · {failed} failed
              {skipped > 0 ? ` · ${skipped} skipped` : ""}
            </p>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}