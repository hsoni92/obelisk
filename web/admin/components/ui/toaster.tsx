"use client";

import {
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastSurface,
  ToastTitle,
  ToastViewport,
} from "@/components/ui/toast";
import { dismiss, getToasts, remove, subscribe } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import { useSyncExternalStore } from "react";

const toastVariants = cva("ring-1", {
  variants: {
    variant: {
      default: "bg-card ring-border",
      success: "bg-success-tint ring-success/25",
      error: "bg-destructive-tint ring-destructive/25",
      info: "bg-link-tint ring-link/25",
      warning: "bg-warning-tint ring-warning/25",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

const titleVariants = cva("text-[13px] font-semibold", {
  variants: {
    variant: {
      default: "text-foreground",
      success: "text-success",
      error: "text-destructive",
      info: "text-link",
      warning: "text-warning",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export function Toaster() {
  const toasts = useSyncExternalStore(subscribe, getToasts, () => []);

  return (
    <ToastProvider duration={5000} label="Notifications">
      {toasts.map((t) => (
        <ToastSurface
          key={t.id}
          open={t.open}
          onOpenChange={(open) => {
            if (!open) {
              dismiss(t.id);
              window.setTimeout(() => remove(t.id), 300);
            }
          }}
          className={cn(toastVariants({ variant: t.variant as VariantProps<typeof toastVariants>["variant"] }))}
        >
          <div className="grid min-w-0 flex-1 gap-0.5">
            <ToastTitle
              className={titleVariants({
                variant: t.variant as VariantProps<typeof titleVariants>["variant"],
              })}
            >
              {t.title}
            </ToastTitle>
            {t.description ? (
              <ToastDescription>{t.description}</ToastDescription>
            ) : null}
          </div>
          <ToastClose />
        </ToastSurface>
      ))}
      <ToastViewport />
    </ToastProvider>
  );
}