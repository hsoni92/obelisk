import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import * as React from "react";

export type SelectOption = { value: string; label: string };

/**
 * Native `<select>` — lightweight for simple option lists (filters, forms).
 * For listbox UX use `@/components/ui/select` (Radix Select).
 */
const NativeSelect = React.forwardRef<
  HTMLSelectElement,
  Omit<React.ComponentProps<"select">, "children"> & {
    options: readonly SelectOption[];
    placeholder?: string;
  }
>(({ className, options, placeholder, ...props }, ref) => (
  <div className={cn("relative w-full", className)}>
    <select
      ref={ref}
      className="field w-full appearance-none pr-8 disabled:opacity-50"
      {...props}
    >
      {placeholder && (
        <option value="" disabled>
          {placeholder}
        </option>
      )}
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
    <ChevronDown
      className="pointer-events-none absolute right-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
      aria-hidden
    />
  </div>
));
NativeSelect.displayName = "NativeSelect";

export { NativeSelect };
