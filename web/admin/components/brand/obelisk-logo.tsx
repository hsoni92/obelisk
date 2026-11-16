"use client";

import { motion } from "framer-motion";
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
    <div className="flex items-center overflow-clip">
      <ObeliskMark className="size-8" alt={compact ? "Obelisk" : ""} />
      <motion.div
        initial={false}
        animate={{
          opacity: compact ? 0 : 1,
          width: compact ? 0 : "auto",
          marginLeft: compact ? 0 : 10,
        }}
        transition={{
          opacity: { duration: 0.12, delay: compact ? 0 : 0.14 },
          width: { duration: 0.22, ease: [0.32, 0.72, 0, 1] },
          marginLeft: { duration: 0.22, ease: [0.32, 0.72, 0, 1] },
        }}
        aria-hidden={compact}
        className="whitespace-nowrap leading-tight"
      >
        <p className="text-base font-medium tracking-tight text-foreground">
          Obelisk
        </p>
        {subtitle && (
          <p className="text-eyebrow text-[10px] text-muted-foreground">
            {subtitle}
          </p>
        )}
      </motion.div>
    </div>
  );
}
