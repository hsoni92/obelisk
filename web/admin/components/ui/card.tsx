import { cn } from "@/lib/utils";

export function Card({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "rounded-2xl bg-card text-card-foreground shadow-rb-2",
        className,
      )}
      {...props}
    />
  );
}

export function CardHeader({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div className={cn("flex flex-col gap-1 p-6 pb-0", className)} {...props} />
  );
}

export function CardTitle({ className, ...props }: React.ComponentProps<"h3">) {
  return (
    <h3
      className={cn(
        "text-[15px] font-normal leading-5 line-clamp-2 text-foreground",
        className,
      )}
      {...props}
    />
  );
}

export function CardDescription({
  className,
  ...props
}: React.ComponentProps<"p">) {
  return (
    <p className={cn("text-xs text-muted-foreground", className)} {...props} />
  );
}

export function CardContent({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return <div className={cn("p-6", className)} {...props} />;
}

/** Forest-green feature surface (the hero control card). */
export function FeatureCard({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "rounded-2xl bg-feature p-6 text-feature-foreground shadow-rb-2",
        className,
      )}
      {...props}
    />
  );
}

/** Inset tile for small stats inside a card (heart rate, reps, …). */
export function StatTile({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("rounded-xl bg-muted px-4 py-3", className)}
      {...props}
    />
  );
}

export type CardTone = "neutral" | "lime" | "green" | "blue" | "amber";

const TONES: Record<CardTone, { band: string; tile: string }> = {
  neutral: {
    band: "from-muted/80 via-muted/30",
    tile: "bg-card text-muted-foreground ring-border",
  },
  lime: {
    band: "from-broadcast-tint via-broadcast-tint/40",
    tile: "bg-card text-foreground ring-broadcast/60",
  },
  green: {
    band: "from-success-tint via-success-tint/40",
    tile: "bg-card text-success ring-success/25",
  },
  blue: {
    band: "from-link-tint via-link-tint/40",
    tile: "bg-card text-link ring-link/25",
  },
  amber: {
    band: "from-warning-tint via-warning-tint/40",
    tile: "bg-card text-warning ring-warning/25",
  },
};

/**
 * Shared card header band: soft tone gradient, icon tile, title, description,
 * optional badge and actions. Pass negative margins via `className` to bleed
 * to the card edge (the card needs `overflow-hidden`).
 */
export function CardHeading({
  icon: Icon,
  title,
  description,
  badge,
  actions,
  tone = "neutral",
  as: Title = "h2",
  className,
}: {
  icon?: React.ComponentType<{ className?: string; "aria-hidden"?: boolean }>;
  title: React.ReactNode;
  description?: React.ReactNode;
  badge?: React.ReactNode;
  actions?: React.ReactNode;
  tone?: CardTone;
  as?: "h2" | "h3" | "p";
  className?: string;
}) {
  const t = TONES[tone];
  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-3 border-b border-border/60 bg-gradient-to-r to-transparent px-4 py-3",
        t.band,
        className,
      )}
    >
      {Icon ? (
        <span
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-xl shadow-rb-1 ring-1",
            t.tile,
          )}
        >
          <Icon className="size-4" aria-hidden />
        </span>
      ) : null}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Title className="truncate text-sm font-semibold leading-5 text-foreground">
            {title}
          </Title>
          {badge}
        </div>
        {description ? (
          <div className="text-metadata text-muted-foreground">
            {description}
          </div>
        ) : null}
      </div>
      {actions ? (
        <div className="flex shrink-0 items-center gap-2">{actions}</div>
      ) : null}
    </div>
  );
}
