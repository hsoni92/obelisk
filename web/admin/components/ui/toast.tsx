"use client";

import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import * as ToastPrimitive from "radix-ui/toast";
import * as React from "react";

const ToastProvider = ToastPrimitive.Provider;
const ToastViewport = React.forwardRef<
  React.ComponentRef<typeof ToastPrimitive.Viewport>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Viewport>
>(({ className, ...props }, ref) => (
  <ToastPrimitive.Viewport
    ref={ref}
    className={cn(
      "fixed bottom-0 right-0 z-[100] flex max-h-screen w-full flex-col-reverse gap-2 p-4 sm:max-w-md",
      className,
    )}
    {...props}
  />
));
ToastViewport.displayName = ToastPrimitive.Viewport.displayName;

const Toast = ToastPrimitive.Root;

const ToastAction = React.forwardRef<
  React.ComponentRef<typeof ToastPrimitive.Action>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Action>
>(({ className, ...props }, ref) => (
  <ToastPrimitive.Action
    ref={ref}
    className={cn(
      "inline-flex h-8 shrink-0 items-center justify-center rounded-full bg-card px-3 text-xs font-medium ring-1 ring-border",
      className,
    )}
    {...props}
  />
));
ToastAction.displayName = ToastPrimitive.Action.displayName;

const ToastClose = React.forwardRef<
  React.ComponentRef<typeof ToastPrimitive.Close>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Close>
>(({ className, ...props }, ref) => (
  <ToastPrimitive.Close
    ref={ref}
    className={cn(
      "absolute right-2 top-2 rounded-full p-1 text-muted-foreground hover:bg-surface",
      className,
    )}
    toast-close=""
    {...props}
  >
    <X className="size-3.5" />
  </ToastPrimitive.Close>
));
ToastClose.displayName = ToastPrimitive.Close.displayName;

const ToastTitle = ({
  className,
  ...props
}: React.ComponentProps<typeof ToastPrimitive.Title>) => (
  <ToastPrimitive.Title
    className={cn("text-[13px] font-semibold", className)}
    {...props}
  />
);

const ToastDescription = ({
  className,
  ...props
}: React.ComponentProps<typeof ToastPrimitive.Description>) => (
  <ToastPrimitive.Description
    className={cn("text-metadata text-muted-foreground", className)}
    {...props}
  />
);

function ToastSurface({
  className,
  ...props
}: React.ComponentProps<typeof ToastPrimitive.Root>) {
  return (
    <Toast
      className={cn(
        "group relative flex w-full items-center gap-3 overflow-hidden rounded-xl border border-border bg-card p-4 pr-8 shadow-rb-2",
        className,
      )}
      {...props}
    />
  );
}

export {
  Toast,
  ToastAction,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastSurface,
  ToastTitle,
  ToastViewport,
};