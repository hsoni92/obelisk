"use client";

import { focusRingClass } from "@/components/ui/radix-styles";
import { cn } from "@/lib/utils";
import * as ToolbarPrimitive from "radix-ui/toolbar";
import * as React from "react";

const Toolbar = React.forwardRef<
  React.ComponentRef<typeof ToolbarPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ToolbarPrimitive.Root>
>(({ className, orientation = "horizontal", ...props }, ref) => (
  <ToolbarPrimitive.Root
    ref={ref}
    orientation={orientation}
    className={cn(
      "flex gap-1 rounded-xl bg-muted p-1",
      orientation === "vertical" && "flex-col",
      className,
    )}
    {...props}
  />
));
Toolbar.displayName = ToolbarPrimitive.Root.displayName;

const ToolbarButton = React.forwardRef<
  React.ComponentRef<typeof ToolbarPrimitive.Button>,
  React.ComponentPropsWithoutRef<typeof ToolbarPrimitive.Button>
>(({ className, ...props }, ref) => (
  <ToolbarPrimitive.Button
    ref={ref}
    className={cn(
      "inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-2 text-[13px] font-medium text-foreground hover:bg-card",
      focusRingClass,
      className,
    )}
    {...props}
  />
));
ToolbarButton.displayName = ToolbarPrimitive.Button.displayName;

const ToolbarSeparator = ({
  className,
  ...props
}: React.ComponentProps<typeof ToolbarPrimitive.Separator>) => (
  <ToolbarPrimitive.Separator
    className={cn("mx-1 w-px self-stretch bg-border", className)}
    {...props}
  />
);

const ToolbarToggleGroup = ToolbarPrimitive.ToggleGroup;
const ToolbarToggleItem = ({
  className,
  ...props
}: React.ComponentProps<typeof ToolbarPrimitive.ToggleItem>) => (
  <ToolbarPrimitive.ToggleItem
    className={cn(
      "inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-2 text-[13px] font-medium",
      "text-foreground hover:bg-card data-[state=on]:bg-card data-[state=on]:text-foreground",
      focusRingClass,
      className,
    )}
    {...props}
  />
);

const ToolbarLink = ({
  className,
  ...props
}: React.ComponentProps<typeof ToolbarPrimitive.Link>) => (
  <ToolbarPrimitive.Link
    className={cn(
      "inline-flex h-8 items-center rounded-lg px-2 text-[13px] font-medium text-link hover:underline",
      focusRingClass,
      className,
    )}
    {...props}
  />
);

export {
  Toolbar,
  ToolbarButton,
  ToolbarLink,
  ToolbarSeparator,
  ToolbarToggleGroup,
  ToolbarToggleItem,
};