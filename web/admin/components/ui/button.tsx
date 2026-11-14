import * as React from "react";
import { Slot } from "radix-ui/slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-broadcast text-white shadow-rb-3 hover:bg-broadcast-hover rounded-full h-8 px-3.5 text-[13px] font-medium",
        subscribed:
          "bg-surface text-foreground hover:bg-surface/80 rounded-full h-8 px-3.5 text-[13px]",
        outline:
          "bg-card text-foreground ring-1 ring-border hover:bg-muted rounded-full h-8 px-3.5 text-[13px] font-medium",
        icon: "size-8 rounded-full text-muted-foreground hover:bg-surface bg-transparent",
        broadcast:
          "bg-broadcast text-white shadow-rb-3 hover:bg-broadcast-hover rounded-full h-8 px-3.5 text-[13px] font-medium",
        link: "text-link font-medium underline-offset-4 hover:underline p-0 h-auto",
      },
      size: {
        default: "h-8 px-3.5",
        sm: "h-7 px-3 text-xs",
        lg: "h-9 px-4 text-sm",
        icon: "size-8",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends
    React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
