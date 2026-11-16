"use client";

import { cn } from "@/lib/utils";
import * as FormPrimitive from "radix-ui/form";
import * as React from "react";

const Form = FormPrimitive.Root;
const FormField = FormPrimitive.Field;
const FormLabel = ({
  className,
  ...props
}: React.ComponentProps<typeof FormPrimitive.Label>) => (
  <FormPrimitive.Label
    className={cn("text-label text-muted-foreground", className)}
    {...props}
  />
);
const FormControl = FormPrimitive.Control;
const FormMessage = ({
  className,
  ...props
}: React.ComponentProps<typeof FormPrimitive.Message>) => (
  <FormPrimitive.Message
    className={cn("text-xs font-medium text-destructive", className)}
    {...props}
  />
);
const FormValidityState = FormPrimitive.ValidityState;
const FormSubmit = FormPrimitive.Submit;

export {
  Form,
  FormControl,
  FormField,
  FormLabel,
  FormMessage,
  FormSubmit,
  FormValidityState,
};
