"use client";

import * as LabelPrimitive from "radix-ui/label";
import { cn } from "@/lib/utils";

const Label = ({
  className,
  ...props
}: React.ComponentProps<typeof LabelPrimitive.Root>) => (
  <LabelPrimitive.Root
    className={cn("text-label text-muted-foreground", className)}
    {...props}
  />
);

export { Label };
