import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2 py-px text-[10px] font-semibold uppercase tracking-wider",
  {
    variants: {
      variant: {
        default: "bg-surface text-foreground",
        live: "bg-broadcast-tint text-broadcast",
        success: "bg-success-tint text-success",
        warning: "bg-warning-tint text-warning",
        info: "bg-link-tint text-link",
        outline: "border border-border text-muted-foreground bg-card",
        duration:
          "bg-black/60 text-white text-[11px] font-medium px-2 py-1 normal-case tracking-normal",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export function Badge({
  className,
  variant,
  ...props
}: React.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return (
    <span className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}
