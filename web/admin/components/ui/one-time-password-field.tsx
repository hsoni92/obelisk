"use client";

/**
 * Radix experimental OTP field — API may change.
 * @see https://www.radix-ui.com/primitives
 */
import { OneTimePasswordField as OTPModule } from "@/lib/radix-primitives";
import { cn } from "@/lib/utils";
import * as React from "react";

const OneTimePasswordField = OTPModule.Root;
const OneTimePasswordFieldInput = React.forwardRef<
  React.ComponentRef<typeof OTPModule.Input>,
  React.ComponentPropsWithoutRef<typeof OTPModule.Input>
>(({ className, ...props }, ref) => (
  <OTPModule.Input
    ref={ref}
    className={cn("field size-9 text-center font-mono text-sm", className)}
    {...props}
  />
));
OneTimePasswordFieldInput.displayName = "OneTimePasswordFieldInput";

const OneTimePasswordFieldHiddenInput = OTPModule.HiddenInput;

export {
  OneTimePasswordField,
  OneTimePasswordFieldHiddenInput,
  OneTimePasswordFieldInput,
};