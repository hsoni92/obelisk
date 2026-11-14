"use client";

import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

export function FacetRow({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-2 sm:flex-row sm:items-start sm:gap-4", className)}>
      <span className="shrink-0 pt-1 text-eyebrow text-muted-foreground sm:w-20">
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}