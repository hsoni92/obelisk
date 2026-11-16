import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-normal tracking-wide",
  {
    variants: {
      variant: {
        default: "bg-muted text-foreground",
        live: "bg-broadcast text-broadcast-foreground",
        success: "bg-success-tint text-success",
        warning: "bg-warning-tint text-warning",
        info: "bg-link-tint text-link",
        outline: "text-muted-foreground bg-card ring-1 ring-border",
        duration:
          "bg-black/60 text-white text-[11px] px-2 py-1 tracking-normal",
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
