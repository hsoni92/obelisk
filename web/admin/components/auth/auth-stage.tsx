"use client";

import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import { ObeliskScene } from "@/components/auth/obelisk-scene";
import { AnimatedObeliskMark } from "@/components/brand/animated-obelisk";
import { ObeliskMark } from "@/components/brand/obelisk-logo";
import {
  motion,
  useMotionTemplate,
  useMotionValue,
  } from "framer-motion";
import { CloudUpload, Layers, ShieldCheck, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

const EASE = [0.32, 0.72, 0, 1] as const;

const POINTS: { icon: LucideIcon; title: string; text: string }[] = [
  {
    icon: ShieldCheck,
    title: "Encrypted",
    text: "Secrets are encrypted before they reach GitHub.",
  },
  {
    icon: CloudUpload,
    title: "Always in sync",
    text: "Changes are pushed to your private repository.",
  },
  {
    icon: Layers,
    title: "Restore anywhere",
    text: "Rebuild this appliance on a new machine.",
  },
];

/**
 * Split-screen frame for sign-in: an animated hero on the left (desktop) and
 * the form on the right. On small screens the hero shrinks to the animated mark.
 */
export function AuthStage({ children }: { children: ReactNode }) {
  const reduce = useSafeReducedMotion();
  const x = useMotionValue(420);
  const y = useMotionValue(260);
  const spot = useMotionTemplate`radial-gradient(360px circle at ${x}px ${y}px, rgb(215 242 73 / 0.13), transparent 65%)`;

  const rise = (delay: number) =>
    reduce
      ? {}
      : {
          initial: { opacity: 0, y: 14 },
          animate: { opacity: 1, y: 0 },
          transition: { duration: 0.6, delay, ease: EASE },
        };

  return (
    <div className="min-h-screen p-3 text-foreground lg:grid lg:grid-cols-[minmax(0,1.12fr)_minmax(440px,0.88fr)] lg:gap-3 lg:p-4">
      <aside
        className="relative isolate hidden flex-col overflow-hidden rounded-[28px] text-white ring-1 ring-black/5 dark:ring-white/10 lg:sticky lg:top-4 lg:flex lg:h-[calc(100vh-2rem)] lg:min-h-[600px]"
        style={{
          background:
            "linear-gradient(155deg, #26382d 0%, #16221b 46%, #0c140f 100%)",
        }}
        onPointerMove={(e) => {
          const r = e.currentTarget.getBoundingClientRect();
          x.set(e.clientX - r.left);
          y.set(e.clientY - r.top);
        }}
      >
        <div
          aria-hidden
          className="absolute inset-0 -z-10"
          style={{
            background:
              "radial-gradient(60% 50% at 82% 0%, rgb(215 242 73 / 0.16), transparent 70%), radial-gradient(52% 46% at 0% 100%, rgb(143 196 162 / 0.2), transparent 70%)",
          }}
        />
        <div
          aria-hidden
          className="absolute inset-0 -z-10 opacity-40"
          style={{
            backgroundImage:
              "radial-gradient(rgb(255 255 255 / 0.16) 1px, transparent 1px)",
            backgroundSize: "22px 22px",
            maskImage: "radial-gradient(70% 60% at 50% 42%, #000 30%, transparent 100%)",
            WebkitMaskImage:
              "radial-gradient(70% 60% at 50% 42%, #000 30%, transparent 100%)",
          }}
        />
        {!reduce && (
          <motion.div
            aria-hidden
            className="pointer-events-none absolute inset-0 -z-10"
            style={{ background: spot }}
          />
        )}

        <motion.header
          {...rise(0.05)}
          className="relative z-10 flex items-center justify-between px-8 pt-7"
        >
          <div className="flex items-center gap-2.5">
            <ObeliskMark className="size-7" alt="" />
            <div className="leading-tight">
              <p className="text-base font-medium tracking-tight">Obelisk</p>
              <p className="text-eyebrow text-[10px] text-white/50">Admin console</p>
            </div>
          </div>
          <span className="text-eyebrow rounded-full bg-white/8 px-3 py-1.5 text-[10px] text-white/70 ring-1 ring-white/10">
            Secure sign-in
          </span>
        </motion.header>

        {/* The scene is taller than its slot on purpose: its floor fades out behind the headline. */}
        <div className="relative min-h-0 flex-1">
          <ObeliskScene className="absolute left-1/2 top-[-6%] h-[138%] w-auto max-w-none -translate-x-1/2" />
        </div>

        <div className="relative z-10 px-8 pb-8">
          <motion.h2
            {...rise(0.9)}
            className="text-[34px] font-light leading-[38px] tracking-[-0.03em]"
          >
            Your appliance,
            <br />
            sealed and in sync.
          </motion.h2>
          <motion.p {...rise(1.0)} className="mt-3 max-w-md text-body text-white/65">
            Sign in with the GitHub account that holds this appliance&apos;s encrypted
            blueprint.
          </motion.p>
          <ul className="mt-6 hidden gap-3 sm:grid-cols-3 [@media(min-height:780px)]:grid">
            {POINTS.map(({ icon: Icon, title, text }, i) => (
              <motion.li
                key={title}
                {...rise(1.1 + i * 0.09)}
                className="rounded-2xl bg-white/[0.06] p-3.5 ring-1 ring-white/10 backdrop-blur-sm"
              >
                <span className="flex size-8 items-center justify-center rounded-full bg-[#d7f249]/15 text-[#e6f97a]">
                  <Icon className="size-4" strokeWidth={1.75} />
                </span>
                <p className="mt-2.5 text-[13px] font-medium">{title}</p>
                <p className="mt-0.5 text-[12px] leading-[17px] text-white/55">{text}</p>
              </motion.li>
            ))}
          </ul>
        </div>
      </aside>

      <main className="relative flex flex-col items-center justify-center px-1 py-8 sm:px-6 lg:py-10">
        <div className="w-full max-w-[420px]">
          <motion.div
            {...rise(0)}
            className="mb-7 flex flex-col items-center text-center lg:hidden"
          >
            <div className="w-28">
              <AnimatedObeliskMark decorative />
            </div>
            <p className="-mt-1 text-lg font-medium tracking-tight">Obelisk</p>
            <p className="text-eyebrow text-[10px] text-muted-foreground">Admin console</p>
          </motion.div>
          {children}
        </div>
      </main>
    </div>
  );
}
