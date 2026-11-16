"use client";

import * as ToggleGroupPrimitive from "radix-ui/toggle-group";
import { cn } from "@/lib/utils";

const ToggleGroup = ToggleGroupPrimitive.Root;

const ToggleGroupItem = ({
  className,
  ...props
}: React.ComponentProps<typeof ToggleGroupPrimitive.Item>) => (
  <ToggleGroupPrimitive.Item
    className={cn(
      "shrink-0 rounded-full px-4 py-1.5 text-[13px] font-normal transition-colors",
      "text-foreground bg-muted hover:bg-surface",
      "data-[state=on]:bg-broadcast data-[state=on]:text-broadcast-foreground",
      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
      className,
    )}
    {...props}
  />
);

function ToggleGroupScroll({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex gap-3 overflow-x-auto pb-1 scrollbar-none",
        className,
      )}
    >
      {children}
    </div>
  );
}

export { ToggleGroup, ToggleGroupItem, ToggleGroupScroll };
