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
      "inline-flex items-center justify-center rounded-full px-3 py-1 text-xs font-medium transition-colors",
      "bg-card text-foreground hover:bg-surface",
      "data-[state=on]:bg-foreground data-[state=on]:text-background",
      focusRingClass,
      className,
    )}
    {...props}
  />
));
Toggle.displayName = TogglePrimitive.Root.displayName;

export { Toggle };
