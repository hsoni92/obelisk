"use client";

import { focusRingClass } from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import * as TogglePrimitive from "radix-ui/toggle";
import * as React from "react";

const Toggle = React.forwardRef<
  React.ComponentRef<typeof TogglePrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof TogglePrimitive.Root>
>(({ className, ...props }, ref) => (
  <TogglePrimitive.Root
    ref={ref}
    className={cn(
      "inline-flex items-center justify-center rounded-full px-4 py-1.5 text-[13px] font-normal transition-colors",
      "bg-muted text-foreground hover:bg-surface",
      "data-[state=on]:bg-broadcast data-[state=on]:text-broadcast-foreground",
      focusRingClass,
      className,
    )}
    {...props}
  />
));
Toggle.displayName = TogglePrimitive.Root.displayName;

export { Toggle };
