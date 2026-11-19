import * as React from "react";
import { Slot } from "radix-ui/slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap font-normal transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-broadcast text-broadcast-foreground hover:bg-broadcast-hover rounded-full h-10 px-5 text-sm font-medium",
        subscribed:
          "bg-muted text-foreground hover:bg-surface rounded-full h-10 px-5 text-sm",
        ink: "bg-ink text-ink-foreground hover:bg-ink/85 rounded-full h-10 px-5 text-sm font-medium",
        outline:
          "bg-card text-foreground ring-1 ring-border hover:bg-muted rounded-full h-10 px-5 text-sm",
        icon: "rounded-full p-0 text-foreground bg-card shadow-rb-1 hover:bg-muted [&_svg]:size-4",
        broadcast:
          "bg-broadcast text-broadcast-foreground shadow-rb-3 hover:bg-broadcast-hover rounded-full h-10 px-5 text-sm font-medium",
        link: "text-link font-medium underline-offset-4 hover:underline p-0 h-auto",
      },
      size: {
        default: "h-10 px-5",
        sm: "h-8 px-4 text-[13px]",
        lg: "h-12 px-6 text-[15px]",
        icon: "size-10 p-0 gap-0",
      },
    },
    compoundVariants: [
      {
        variant: "outline",
        size: "icon",
        class: "p-0 ring-1 ring-border",
      },
      {
        variant: "icon",
        size: "sm",
        class: "size-8 p-0",
      },
    ],
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
