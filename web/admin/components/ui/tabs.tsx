"use client";

import * as TabsPrimitive from "radix-ui/tabs";
import { cn } from "@/lib/utils";

const Tabs = TabsPrimitive.Root;

const TabsList = ({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List>) => (
  <TabsPrimitive.List
    className={cn(
      "flex gap-2 overflow-x-auto bg-surface/50 py-2 px-1 scrollbar-none",
      className,
    )}
    {...props}
  />
);

const TabsTrigger = ({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger>) => (
  <TabsPrimitive.Trigger
    className={cn(
      "shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-colors",
      "text-foreground bg-card hover:bg-surface",
      "data-[state=active]:bg-foreground data-[state=active]:text-background",
      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
      className,
    )}
    {...props}
  />
);

const TabsContent = ({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) => (
  <TabsPrimitive.Content
    className={cn("mt-6 focus:outline-none", className)}
    {...props}
  />
);

export { Tabs, TabsList, TabsTrigger, TabsContent };
