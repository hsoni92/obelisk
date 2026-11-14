"use client";

/**
 * Radix experimental password toggle field — API may change.
 */
import { PasswordToggleField as PasswordFieldModule } from "@/lib/radix-primitives";
import { cn } from "@/lib/utils";
import * as React from "react";

const PasswordToggleField = PasswordFieldModule.Root;
const PasswordToggleFieldInput = React.forwardRef<
  React.ComponentRef<typeof PasswordFieldModule.Input>,
  React.ComponentPropsWithoutRef<typeof PasswordFieldModule.Input>
>(({ className, ...props }, ref) => (
  <PasswordFieldModule.Input
    ref={ref}
    className={cn("field w-full", className)}
    {...props}
  />
));
PasswordToggleFieldInput.displayName = "PasswordToggleFieldInput";

const PasswordToggleFieldToggle = PasswordFieldModule.Toggle;

export {
  PasswordToggleField,
  PasswordToggleFieldInput,
  PasswordToggleFieldToggle,
};