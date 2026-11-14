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
      "peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent bg-muted transition-colors",
      "data-[state=checked]:bg-broadcast",
      "disabled:cursor-not-allowed disabled:opacity-50",
      focusRingClass,
      className,
    )}
    {...props}
  >
    <SwitchPrimitive.Thumb
      className={cn(
        "pointer-events-none block size-4 rounded-full bg-card shadow-rb-1 ring-0 transition-transform",
        "translate-x-0 data-[state=checked]:translate-x-4",
      )}
    />
  </SwitchPrimitive.Root>
));
Switch.displayName = SwitchPrimitive.Root.displayName;

export { Switch };
