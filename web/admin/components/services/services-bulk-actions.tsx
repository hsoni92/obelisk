"use client";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "@/lib/toast";
import { startStopTarget, type ServiceRow } from "@/lib/service-rows";
import type { ServiceInstance } from "@/lib/api";
import { ChevronDown, Play, Square } from "lucide-react";
import { useMemo } from "react";

export type RowPending = { from: string; to: "start" | "stop" };

export function useBulkActionCounts(rows: ServiceRow[], selected: Set<string>) {
  return useMemo(() => {
    let start = 0;
    let stop = 0;
    for (const row of rows) {
      if (!selected.has(row.rowKey)) continue;
      const t = startStopTarget(row.state);
      if (t === "start") start += 1;
      if (t === "stop") stop += 1;
    }
    return { start, stop };
  }, [rows, selected]);
}

export async function runServicesBulkAction(
  rows: ServiceRow[],
  selected: Set<string>,
  action: "start" | "stop",
  onToggle: (inst: ServiceInstance, enabled: boolean) => Promise<boolean>,
  markPending: (key: string, pending: RowPending) => void,
  onClearSelection: () => void,
): Promise<void> {
  const chosen = rows.filter((r) => selected.has(r.rowKey));
  const eligible = chosen.filter((r) => startStopTarget(r.state) === action);
  if (eligible.length === 0) {
    toast.warning(
      action === "start" ? "Nothing to start" : "Nothing to stop",
      chosen.length === 0
        ? "Select one or more services in the table first."
        : "Selected services are still starting, stopping, or already in the target state.",
    );
    return;
  }
  let ok = 0;
  let fail = 0;
  for (const row of eligible) {
    markPending(row.rowKey, { from: row.state, to: action });
    const success = await onToggle(row.instance, action === "start");
    if (success) ok += 1;
    else fail += 1;
  }
  if (fail === 0) {
    toast.success(
      action === "start" ? "Start queued" : "Stop queued",
      `${ok} instance${ok === 1 ? "" : "s"}.`,
    );
    onClearSelection();
  } else if (ok === 0) {
    toast.error("Action failed", "Could not update the selected instances.");
  } else {
    toast.warning("Partially applied", `${ok} updated, ${fail} failed.`);
  }
}

type ServicesBulkActionsMenuProps = {
  selectedCount: number;
  startCount: number;
  stopCount: number;
  disabled?: boolean;
  busy?: boolean;
  onStart: () => void;
  onStop: () => void;
  onClear: () => void;
};

export function ServicesBulkActionsMenu({
  selectedCount,
  startCount,
  stopCount,
  disabled = false,
  busy = false,
  onStart,
  onStop,
  onClear,
}: ServicesBulkActionsMenuProps) {
  const locked = disabled || busy;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={locked || selectedCount === 0}
          aria-label={
            selectedCount > 0
              ? `Actions, ${selectedCount} selected`
              : "Actions, select services first"
          }
        >
          Actions
          {selectedCount > 0 && (
            <span className="ml-1 rounded-full bg-broadcast-tint px-1.5 text-[10px] font-semibold text-foreground">
              {selectedCount}
            </span>
          )}
          <ChevronDown className="size-3.5 opacity-70" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuLabel className="font-normal text-muted-foreground">
          {selectedCount === 0
            ? "Select rows in the table"
            : `${selectedCount} selected`}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={locked || startCount === 0}
          onSelect={() => onStart()}
        >
          <Play className="size-3.5" aria-hidden />
          Start{startCount > 0 ? ` (${startCount})` : ""}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={locked || stopCount === 0}
          onSelect={() => onStop()}
        >
          <Square className="size-3.5" aria-hidden />
          Stop{stopCount > 0 ? ` (${stopCount})` : ""}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={locked || selectedCount === 0}
          onSelect={() => onClear()}
        >
          Clear selection
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}