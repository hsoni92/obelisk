"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";

export type ActiveFilterChip = {
  id: string;
  label: string;
  onRemove: () => void;
};

export function FilterActiveSummary({
  chips,
  activeCount,
  onClearAll,
  className,
}: {
  chips: ActiveFilterChip[];
  activeCount: number;
  onClearAll: () => void;
  className?: string;
}) {
  if (activeCount === 0) return null;

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 rounded-xl border border-border bg-card px-3 py-2 shadow-rb-2",
        className,
      )}
    >
      <span className="text-eyebrow text-muted-foreground">
        {activeCount} active filter{activeCount === 1 ? "" : "s"}
      </span>
      <div className="flex flex-wrap gap-1.5">
        {chips.map((chip) => (
          <button
            key={chip.id}
            type="button"
            onClick={chip.onRemove}
            className="inline-flex items-center gap-1 rounded-full bg-muted px-2.5 py-0.5 text-[11px] font-medium text-foreground hover:bg-surface"
          >
            {chip.label}
            <X className="size-3 opacity-60" aria-hidden />
            <span className="sr-only">Remove {chip.label}</span>
          </button>
        ))}
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="ml-auto shrink-0"
        onClick={onClearAll}
      >
        Clear all
      </Button>
    </div>
  );
}