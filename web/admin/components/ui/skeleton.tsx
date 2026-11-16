import { cn } from "@/lib/utils";

/** Placeholder block. Pulses (unless the user prefers reduced motion). Decorative only. */
export function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      aria-hidden
      className={cn(
        "rounded-lg bg-surface motion-safe:animate-pulse",
        className,
      )}
      {...props}
    />
  );
}

/**
 * Wraps a skeleton region so assistive tech announces one "Loading" status
 * instead of reading dozens of empty blocks.
 */
export function SkeletonRegion({
  label = "Loading",
  className,
  children,
  ...props
}: React.ComponentProps<"div"> & { label?: string }) {
  return (
    <div role="status" aria-busy="true" className={className} {...props}>
      <span className="sr-only">{label}…</span>
      {children}
    </div>
  );
}
