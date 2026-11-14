"use client";

import { focusRingClass } from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import * as CheckboxPrimitive from "radix-ui/checkbox";
import * as React from "react";

const Checkbox = React.forwardRef<
  React.ComponentRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      "peer size-4 shrink-0 rounded-sm border border-border bg-card shadow-rb-1",
      "data-[state=checked]:border-broadcast data-[state=checked]:bg-broadcast data-[state=checked]:text-white",
      "disabled:cursor-not-allowed disabled:opacity-50",
      focusRingClass,
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator className="flex items-center justify-center text-current">
      <Check className="size-3" strokeWidth={3} />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
));
Checkbox.displayName = CheckboxPrimitive.Root.displayName;

export { Checkbox };
