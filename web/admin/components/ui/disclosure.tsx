"use client";

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { focusRingClass } from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import * as React from "react";

type Props = {
  title: React.ReactNode;
  /** Optional leading icon. */
  icon?: React.ReactNode;
  /** Optional right-aligned hint (e.g. a count or badge). */
  meta?: React.ReactNode;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
  contentClassName?: string;
  children: React.ReactNode;
};

/** Shared expandable section: bordered, rounded, smooth height animation. */
export function Disclosure({
  title,
  icon,
  meta,
  defaultOpen,
  open,
  onOpenChange,
  className,
  contentClassName,
  children,
}: Props) {
  return (
    <Collapsible
      defaultOpen={defaultOpen}
      open={open}
      onOpenChange={onOpenChange}
      className={cn(
        "group/disclosure rounded-xl border border-border/70 bg-muted/20 transition-colors data-[state=open]:bg-muted/30",
        className,
      )}
    >
      <CollapsibleTrigger
        className={cn(
          "flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-xs font-medium text-foreground transition-colors hover:bg-muted/50",
          focusRingClass,
        )}
      >
        <ChevronRight
          className="size-3.5 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/disclosure:rotate-90"
          aria-hidden
        />
        {icon ? (
          <span className="shrink-0 text-muted-foreground">{icon}</span>
        ) : null}
        <span className="min-w-0 flex-1 truncate">{title}</span>
        {meta ? (
          <span className="shrink-0 text-[11px] font-normal text-muted-foreground">
            {meta}
          </span>
        ) : null}
      </CollapsibleTrigger>
      <CollapsibleContent
        forceMount
        className="grid transition-[grid-template-rows] duration-200 ease-out data-[state=closed]:grid-rows-[0fr] data-[state=open]:grid-rows-[1fr]"
      >
        <div className="min-h-0 overflow-hidden">
          <div
            className={cn(
              "border-t border-border/60 px-3 py-3 group-data-[state=closed]/disclosure:invisible",
              contentClassName,
            )}
          >
            {children}
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
