import { cn } from "@/lib/utils";
import Image from "next/image";

type ObeliskMarkProps = {
  className?: string;
  alt?: string;
};

export function ObeliskMark({ className, alt = "Obelisk" }: ObeliskMarkProps) {
  return (
    <Image
      src="/brand/obelisk-logo.svg"
      alt={alt}
      width={248}
      height={270}
      unoptimized
      priority
      className={cn("size-8 shrink-0 object-contain select-none", className)}
    />
  );
}

export function ObeliskWordmark({
  compact = false,
  subtitle = "Admin console",
}: {
  compact?: boolean;
  subtitle?: string;
}) {
  return (
    <div className={cn("flex items-center gap-2.5", compact && "justify-center")}>
      <ObeliskMark className="size-8" alt={compact ? "Obelisk" : ""} />
      {!compact && (
        <div className="leading-tight">
          <p className="text-base font-medium tracking-tight text-foreground">
            Obelisk
          </p>
          {subtitle && (
            <p className="text-eyebrow text-[10px] text-muted-foreground">
              {subtitle}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
