"use client";

import { focusRingClass } from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import * as SwitchPrimitive from "radix-ui/switch";
import * as React from "react";

const Switch = React.forwardRef<
  React.ComponentRef<typeof SwitchPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitive.Root
    ref={ref}
    className={cn(
      "peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full bg-surface ring-1 ring-inset ring-border transition-colors",
      "data-[state=checked]:bg-switch-on data-[state=checked]:ring-transparent",
      "disabled:cursor-not-allowed disabled:opacity-50",
      focusRingClass,
      className,
    )}
    {...props}
  >
    <SwitchPrimitive.Thumb
      className={cn(
        "pointer-events-none block size-4 rounded-full bg-muted-foreground/70 shadow-rb-1 transition-[transform,background-color]",
        "translate-x-1 data-[state=checked]:translate-x-6 data-[state=checked]:bg-white",
      )}
    />
  </SwitchPrimitive.Root>
));
Switch.displayName = SwitchPrimitive.Root.displayName;

export { Switch };
