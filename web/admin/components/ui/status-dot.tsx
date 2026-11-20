import { cn } from "@/lib/utils";

export type StatusTone = "ok" | "warn" | "bad" | "idle";

const DOT: Record<StatusTone, string> = {
  ok: "bg-success shadow-[0_0_0_3px_color-mix(in_srgb,var(--success)_22%,transparent)]",
  warn: "bg-warning shadow-[0_0_0_3px_color-mix(in_srgb,var(--warning)_22%,transparent)]",
  bad: "bg-destructive shadow-[0_0_0_3px_color-mix(in_srgb,var(--destructive)_22%,transparent)]",
  idle: "bg-muted-foreground/50",
};

/** Brighter palette for the forest-green feature surfaces. */
const DOT_ON_DARK: Record<StatusTone, string> = {
  ok: "bg-broadcast shadow-[0_0_0_3px_rgb(215_242_73/0.22)]",
  warn: "bg-[#f5c27a] shadow-[0_0_0_3px_rgb(245_194_122/0.22)]",
  bad: "bg-[#ff9a90] shadow-[0_0_0_3px_rgb(255_154_144/0.22)]",
  idle: "bg-white/40",
};

/** Maps a service/instance state string to a tone. */
export function stateTone(state?: string): StatusTone {
  switch (state) {
    case "running":
      return "ok";
    case "starting":
    case "stopping":
    case "pending":
      return "warn";
    case "degraded":
    case "failed":
      return "bad";
    default:
      return "idle";
  }
}

export function StatusDot({
  tone,
  className,
  onDark,
  pulse,
}: {
  tone: StatusTone;
  className?: string;
  onDark?: boolean;
  /** Soft breathing animation for in-flight states. */
  pulse?: boolean;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-block size-2 shrink-0 rounded-full",
        (onDark ? DOT_ON_DARK : DOT)[tone],
        pulse && "animate-pulse",
        className,
      )}
    />
  );
}

/** Glowing dot + text. Reusable anywhere a state needs to read at a glance. */
export function StatusLabel({
  tone,
  children,
  onDark,
  pulse,
  className,
  title,
}: {
  tone: StatusTone;
  children: React.ReactNode;
  onDark?: boolean;
  pulse?: boolean;
  className?: string;
  title?: string;
}) {
  return (
    <span
      role="status"
      title={title}
      className={cn("inline-flex items-center gap-2 text-[13px] font-medium", className)}
    >
      <StatusDot tone={tone} onDark={onDark} pulse={pulse} />
      <span className="truncate capitalize">{children}</span>
    </span>
  );
}

/** Convenience wrapper for service states (running, degraded, …). */
export function StateStatus({
  state,
  title,
  className,
}: {
  state: string;
  title?: string;
  className?: string;
}) {
  const tone = stateTone(state);
  return (
    <StatusLabel tone={tone} pulse={tone === "warn"} title={title} className={className}>
      {state}
    </StatusLabel>
  );
}
