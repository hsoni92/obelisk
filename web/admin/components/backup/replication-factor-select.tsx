"use client";

import { SelectField } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { useEffect } from "react";

type Props = {
  id?: string;
  label?: string;
  value: string;
  onChange: (value: string) => void;
  /** Enabled off-site destinations — max selectable redundancy (×N). */
  maxCopies: number;
  className?: string;
};

export function ReplicationFactorSelect({
  id = "replication-factor",
  label = "Replication (min copies)",
  value,
  onChange,
  maxCopies,
  className,
}: Props) {
  const cap = Math.max(0, maxCopies);

  useEffect(() => {
    if (cap === 0) return;
    const n = parseInt(value, 10) || 1;
    if (n > cap) onChange(String(cap));
    else if (n < 1) onChange("1");
  }, [cap, value, onChange]);

  return (
    <div className={cn("space-y-2", className)}>
      <Label htmlFor={id}>{label}</Label>
      {cap === 0 ? (
        <p className="rounded-md border border-dashed border-border px-2 py-1.5 text-sm text-muted-foreground">
          Add at least one enabled destination to choose redundancy.
        </p>
      ) : (
        <SelectField
          id={id}
          value={value}
          onValueChange={onChange}
          options={Array.from({ length: cap }, (_, i) => {
            const n = i + 1;
            return {
              value: String(n),
              label: `×${n} — copy to ${n} destination${n === 1 ? "" : "s"}`,
            };
          })}
        />
      )}
    </div>
  );
}

/** Clamp parsed RF to [1, maxCopies] when destinations exist. */
export function parseReplicationFactor(
  raw: string,
  maxCopies: number,
): number {
  const cap = Math.max(0, maxCopies);
  if (cap === 0) return Math.max(1, parseInt(raw, 10) || 1);
  return Math.min(cap, Math.max(1, parseInt(raw, 10) || 1));
}