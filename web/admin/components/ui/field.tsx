"use client";

import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";

export function Field({
  label,
  className,
  ...inputProps
}: React.ComponentProps<typeof Input> & { label: string }) {
  return (
    <div className={cn("block space-y-1", className)}>
      <Label htmlFor={inputProps.id}>{label}</Label>
      <Input {...inputProps} />
    </div>
  );
}

/** Password field with show/hide toggle (blueprint passphrases, credentials). */
export function FieldPassword({
  label,
  className,
  ...inputProps
}: React.ComponentProps<typeof Input> & { label: string }) {
  const [visible, setVisible] = useState(false);
  return (
    <div className={cn("block space-y-1", className)}>
      <Label htmlFor={inputProps.id}>{label}</Label>
      <div className="relative">
        <Input
          {...inputProps}
          type={visible ? "text" : "password"}
          className="pr-11"
        />
        <Button
          type="button"
          variant="icon"
          size="icon"
          className="absolute right-0.5 top-1/2 size-8 -translate-y-1/2 bg-transparent shadow-none ring-0 hover:bg-muted text-muted-foreground hover:text-foreground"
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? "Hide passphrase" : "Show passphrase"}
          aria-pressed={visible}
        >
          {visible ? (
            <EyeOff className="h-4 w-4" aria-hidden />
          ) : (
            <Eye className="h-4 w-4" aria-hidden />
          )}
        </Button>
      </div>
    </div>
  );
}
