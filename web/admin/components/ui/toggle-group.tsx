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
      "shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-colors",
      "text-foreground bg-card hover:bg-surface",
      "data-[state=on]:bg-foreground data-[state=on]:text-background",
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
