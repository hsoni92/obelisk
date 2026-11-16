"use client";

import {
  focusRingClass,
  popoverContentClass,
} from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import { Check, ChevronDown, Search } from "lucide-react";
import * as CheckboxPrimitive from "radix-ui/checkbox";
import * as PopoverPrimitive from "radix-ui/popover";
import * as React from "react";

export type MultiSelectOption = { value: string; label: string };

/**
 * Radix multi-select: popover trigger summarising the selection, optional
 * search (auto-enabled for long lists), checkbox rows, Select all / Clear.
 */
export function MultiSelect({
  options,
  value,
  onValueChange,
  placeholder = "Any",
  noun = "items",
  searchThreshold = 6,
  className,
  "aria-label": ariaLabel,
}: {
  options: readonly MultiSelectOption[];
  value: string[];
  onValueChange: (next: string[]) => void;
  placeholder?: string;
  noun?: string;
  searchThreshold?: number;
  className?: string;
  "aria-label"?: string;
}) {
  const [q, setQ] = React.useState("");
  const selected = React.useMemo(() => new Set(value), [value]);
  const visible = React.useMemo(() => {
    const needle = q.trim().toLowerCase();
    return needle
      ? options.filter((o) => o.label.toLowerCase().includes(needle))
      : options;
  }, [options, q]);
  const searchable = options.length > searchThreshold;
  const allVisibleSelected =
    visible.length > 0 && visible.every((o) => selected.has(o.value));

  const labels = options
    .filter((o) => selected.has(o.value))
    .map((o) => o.label);
  const summary =
    labels.length === 0
      ? null
      : labels.length <= 2
        ? labels.join(", ")
        : `${labels.length} ${noun} selected`;

  function toggle(v: string) {
    onValueChange(
      selected.has(v) ? value.filter((x) => x !== v) : [...value, v],
    );
  }

  function toggleAllVisible() {
    if (allVisibleSelected) {
      const hide = new Set(visible.map((o) => o.value));
      onValueChange(value.filter((v) => !hide.has(v)));
    } else {
      onValueChange([...new Set([...value, ...visible.map((o) => o.value)])]);
    }
  }

  return (
    <PopoverPrimitive.Root onOpenChange={(o) => !o && setQ("")}>
      <PopoverPrimitive.Trigger
        aria-label={ariaLabel}
        className={cn(
          "field flex h-8 w-full items-center justify-between gap-2 px-3 text-left text-[13px]",
          focusRingClass,
          className,
        )}
      >
        <span className={cn("truncate", !summary && "text-muted-foreground")}>
          {summary ?? placeholder}
        </span>
        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
      </PopoverPrimitive.Trigger>
      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="start"
          sideOffset={4}
          className={cn(
            popoverContentClass,
            "w-[var(--radix-popover-trigger-width)] min-w-[12rem]",
          )}
        >
          {searchable && (
            <div className="flex items-center gap-2 border-b border-border px-3 py-2">
              <Search className="size-3.5 shrink-0 text-muted-foreground" />
              <input
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder={`Search ${noun}…`}
                aria-label={`Search ${noun}`}
                className="w-full bg-transparent text-[13px] outline-none placeholder:text-muted-foreground"
              />
            </div>
          )}
          <ul className="max-h-60 overflow-y-auto p-1" role="group">
            {visible.length === 0 && (
              <li className="px-2 py-3 text-center text-metadata text-muted-foreground">
                No {noun} match “{q}”
              </li>
            )}
            {visible.map((o) => (
              <li key={o.value}>
                <label className="flex cursor-pointer items-center gap-2.5 rounded-lg px-2 py-1.5 text-[13px] hover:bg-muted">
                  <CheckboxPrimitive.Root
                    checked={selected.has(o.value)}
                    onCheckedChange={() => toggle(o.value)}
                    className={cn(
                      "flex size-4 shrink-0 items-center justify-center rounded border border-border bg-card",
                      "data-[state=checked]:border-broadcast data-[state=checked]:bg-broadcast",
                      focusRingClass,
                    )}
                  >
                    <CheckboxPrimitive.Indicator>
                      <Check className="size-3 text-broadcast-foreground" />
                    </CheckboxPrimitive.Indicator>
                  </CheckboxPrimitive.Root>
                  <span className="truncate capitalize">{o.label}</span>
                </label>
              </li>
            ))}
          </ul>
          <div className="flex items-center justify-between border-t border-border px-3 py-1.5 text-xs">
            <span className="text-muted-foreground">
              {value.length > 0 ? `${value.length} selected` : "Any"}
            </span>
            <button
              type="button"
              onClick={toggleAllVisible}
              disabled={visible.length === 0}
              className="font-medium text-link hover:underline disabled:text-muted-foreground"
            >
              {allVisibleSelected ? "Clear" : "Select all"}
            </button>
          </div>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
