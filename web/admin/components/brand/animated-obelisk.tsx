"use client";

import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import { cn } from "@/lib/utils";
import {
  motion,
  useMotionValue,
  useSpring,
  useTransform,
  type Variants,
} from "framer-motion";
import { useEffect, useId } from "react";

type Props = {
  className?: string;
  /** Build the mark piece by piece on mount. */
  assemble?: boolean;
  /** Follow the pointer with a subtle 3D tilt. */
  tilt?: boolean;
  /** Soft lime glow behind the mark. */
  halo?: boolean;
  /** Hide from assistive tech when a label sits next to it. */
  decorative?: boolean;
};

const EASE = [0.32, 0.72, 0, 1] as const;

/** A face slides in from `from`, settling with a light spring. */
function face(from: { x?: number; y?: number }, delay: number): Variants {
  return {
    hidden: { opacity: 0, x: from.x ?? 0, y: from.y ?? 0, scale: 0.94 },
    show: {
      opacity: 1,
      x: 0,
      y: 0,
      scale: 1,
      transition: { type: "spring", stiffness: 110, damping: 17, mass: 0.9, delay },
    },
  };
}

const FADE = (delay: number): Variants => ({
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { duration: 0.7, delay, ease: EASE } },
});

const BOX = { transformBox: "fill-box", transformOrigin: "center" } as const;

/**
 * The Obelisk mark, drawn from the same geometry as /brand/obelisk-logo.svg
 * but split into faces so it can assemble, breathe, catch a moving highlight,
 * run an energy pulse up its edge, and lean toward the cursor. Motion is
 * dropped for people who prefer reduced motion.
 */
export function AnimatedObeliskMark({
  className,
  assemble = true,
  tilt = true,
  halo = true,
  decorative = false,
}: Props) {
  const reduce = useSafeReducedMotion();
  const uid = useId().replace(/:/g, "");
  const id = (name: string) => `${uid}-${name}`;
  const url = (name: string) => `url(#${id(name)})`;
  const play = !reduce;

  const mx = useMotionValue(0);
  const my = useMotionValue(0);
  const sx = useSpring(mx, { stiffness: 80, damping: 16, mass: 0.8 });
  const sy = useSpring(my, { stiffness: 80, damping: 16, mass: 0.8 });
  const rotateY = useTransform(sx, [-1, 1], [-9, 9]);
  const rotateX = useTransform(sy, [-1, 1], [7, -7]);

  useEffect(() => {
    if (!tilt || !play) return;
    const onMove = (e: PointerEvent) => {
      mx.set((e.clientX / window.innerWidth) * 2 - 1);
      my.set((e.clientY / window.innerHeight) * 2 - 1);
    };
    window.addEventListener("pointermove", onMove, { passive: true });
    return () => window.removeEventListener("pointermove", onMove);
  }, [tilt, play, mx, my]);

  const initial = assemble && play ? "hidden" : "show";

  return (
    <motion.div
      className={cn("relative aspect-[296/312] w-full select-none", className)}
      style={play && tilt ? { rotateX, rotateY, transformPerspective: 900 } : undefined}
    >
      {halo && (
        <motion.div
          aria-hidden
          className="pointer-events-none absolute -inset-[18%] rounded-full blur-3xl"
          style={{
            background:
              "radial-gradient(closest-side, rgb(215 242 73 / 0.5), rgb(215 242 73 / 0.12) 55%, transparent 75%)",
          }}
          animate={play ? { opacity: [0.55, 0.95, 0.55], scale: [1, 1.07, 1] } : undefined}
          transition={{ duration: 5.2, repeat: Infinity, ease: "easeInOut" }}
        />
      )}

      <motion.div
        className="absolute inset-0"
        animate={play ? { y: [0, -7, 0] } : undefined}
        transition={{ duration: 5.2, repeat: Infinity, ease: "easeInOut", delay: 1.2 }}
      >
        <motion.svg
          viewBox="-24 -18 296 312"
          className="size-full overflow-visible"
          role={decorative ? undefined : "img"}
          aria-label={decorative ? undefined : "Obelisk"}
          aria-hidden={decorative || undefined}
          initial={initial}
          animate="show"
        >
          <defs>
            <linearGradient id={id("leftDark")} x1="0.3" y1="0" x2="0.7" y2="1">
              <stop offset="0" stopColor="#6f9480" />
              <stop offset="0.5" stopColor="#456352" />
              <stop offset="1" stopColor="#18261d" />
            </linearGradient>
            <linearGradient id={id("leftBevel")} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#b4cdbb" />
              <stop offset="0.55" stopColor="#7fa08c" />
              <stop offset="1" stopColor="#3b5546" />
            </linearGradient>
            <linearGradient id={id("centerLeft")} x1="0.5" y1="0" x2="0.2" y2="1">
              <stop offset="0" stopColor="#8fb09b" />
              <stop offset="0.38" stopColor="#4d6b5a" />
              <stop offset="0.78" stopColor="#2b4335" />
              <stop offset="1" stopColor="#142019" />
            </linearGradient>
            <linearGradient id={id("centerRight")} x1="0.2" y1="0" x2="0.8" y2="1">
              <stop offset="0" stopColor="#eafa78" />
              <stop offset="0.4" stopColor="#d7f249" />
              <stop offset="1" stopColor="#9fbf26" />
            </linearGradient>
            <linearGradient id={id("centerHighlight")} x1="0.2" y1="0" x2="0.8" y2="1">
              <stop offset="0" stopColor="#f8fdb0" />
              <stop offset="0.6" stopColor="#f0fba8" />
              <stop offset="1" stopColor="#dcf55a" />
            </linearGradient>
            <linearGradient id={id("centerBand")} x1="1" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#e6f97a" />
              <stop offset="1" stopColor="#b5d530" />
            </linearGradient>
            <linearGradient id={id("rightSide")} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#5f8370" />
              <stop offset="0.55" stopColor="#3a5546" />
              <stop offset="1" stopColor="#16231b" />
            </linearGradient>
            <linearGradient id={id("barTop")} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stopColor="#f2fc9a" />
              <stop offset="1" stopColor="#d7f249" />
            </linearGradient>
            <linearGradient id={id("barMid")} x1="0" y1="0" x2="1" y2="1">
              <stop offset="0" stopColor="#c9e748" />
              <stop offset="1" stopColor="#8fb92f" />
            </linearGradient>
            <linearGradient id={id("barLow")} x1="0" y1="0" x2="0.6" y2="1">
              <stop offset="0" stopColor="#6f9480" />
              <stop offset="1" stopColor="#2f4a3a" />
            </linearGradient>
            <linearGradient id={id("baseLeft")} x1="0.7" y1="0" x2="0.2" y2="1">
              <stop offset="0" stopColor="#34503f" />
              <stop offset="1" stopColor="#121c15" />
            </linearGradient>
            <linearGradient id={id("baseRight")} x1="0.2" y1="0" x2="0.8" y2="1">
              <stop offset="0" stopColor="#e9f98a" />
              <stop offset="0.5" stopColor="#bfe03a" />
              <stop offset="1" stopColor="#6f9a2a" />
            </linearGradient>
            {/* Moving highlight: a soft diagonal band of light. */}
            <linearGradient id={id("sheen")} x1="0" y1="0" x2="1" y2="0">
              <stop offset="0" stopColor="#fff" stopOpacity="0" />
              <stop offset="0.5" stopColor="#fff" stopOpacity="0.62" />
              <stop offset="1" stopColor="#fff" stopOpacity="0" />
            </linearGradient>
            <radialGradient id={id("ground")} cx="0.5" cy="0.5" r="0.5">
              <stop offset="0" stopColor="#0b120e" stopOpacity="0.38" />
              <stop offset="1" stopColor="#0b120e" stopOpacity="0" />
            </radialGradient>
            <clipPath id={id("clipCentral")}>
              <path d="M124.5 11.5 L93 48 L81 184 L125 158 Z" />
              <path d="M124.5 11.5 L156.5 48 L168 184 L125 158 Z" />
            </clipPath>
            <filter id={id("glow")} x="-40%" y="-40%" width="180%" height="180%">
              <feGaussianBlur stdDeviation="2.2" result="b" />
              <feMerge>
                <feMergeNode in="b" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          {/* Ground shadow */}
          <motion.ellipse
            cx="125"
            cy="262"
            rx="86"
            ry="11"
            fill={url("ground")}
            variants={FADE(0.1)}
            animate={play ? { opacity: [0.75, 1, 0.75], rx: [86, 80, 86] } : undefined}
            transition={{ duration: 5.2, repeat: Infinity, ease: "easeInOut", delay: 1.2 }}
          />

          {/* Front base */}
          <motion.g variants={face({ y: 30 }, 0.05)} style={BOX}>
            <path d="M125 169 L80 195.5 L77 231 L125 257 Z" fill={url("baseLeft")} />
            <path d="M125 169 L170 195.5 L173 231 L125 257 Z" fill={url("baseRight")} />
          </motion.g>

          {/* Left tower */}
          <motion.g variants={face({ x: -30 }, 0.22)} style={BOX}>
            <path d="M31 128 L67 106 L67 227 L31 207 Z" fill={url("leftDark")} />
            <path d="M67 106 L78 113 L68 227 L67 227 Z" fill={url("leftBevel")} />
          </motion.g>

          {/* Right tower */}
          <motion.g variants={face({ x: 30 }, 0.34)} style={BOX}>
            <path d="M169.5 89 L182 81 L182 108 L171.5 112.5 Z" fill={url("rightSide")} />
            <path d="M182 81 L218 103 L218 128 L182 108 Z" fill={url("barTop")} />
            <path d="M172.5 121 L182 116 L182 140 L174 143 Z" fill={url("rightSide")} />
            <path d="M182 116 L218 136 L218 160.5 L182 140 Z" fill={url("barMid")} />
            <path
              d="M175 152 L182 148 L218 169 L218 207 L182 226.5 L177 174 L182 171 L200 179 L200 173 L182 163 L175.5 164 Z"
              fill={url("barLow")}
            />
            <path d="M175 152 L182 148 L182 163 L175.5 164 Z" fill={url("rightSide")} />
            <path d="M177 174 L182 171 L182 226.5 Z" fill={url("rightSide")} />
          </motion.g>

          {/* Central obelisk: drops in from above */}
          <motion.g variants={face({ y: -38 }, 0.48)} style={BOX}>
            <path d="M124.5 11.5 L93 48 L81 184 L125 158 Z" fill={url("centerLeft")} />
            <path d="M124.5 11.5 L156.5 48 L168 184 L125 158 Z" fill={url("centerRight")} />
            <motion.g variants={FADE(1.0)}>
              <path
                d="M124.5 11.5 L156.5 48 L158.5 74 L137 54 L137 127 L125 135 L125 11.5 Z"
                fill={url("centerHighlight")}
              />
              <path d="M125 120 L125 135 L84.5 159.5 L86 144 Z" fill={url("centerBand")} />
            </motion.g>

            {play && (
              <g clipPath={url("clipCentral")} aria-hidden>
                <motion.path
                  d="M-8 0 L26 0 L-2 200 L-36 200 Z"
                  fill={url("sheen")}
                  initial={{ x: 40, y: 0 }}
                  animate={{ x: [40, 190] }}
                  transition={{
                    duration: 1.9,
                    delay: 2.1,
                    repeat: Infinity,
                    repeatDelay: 4.6,
                    ease: [0.4, 0, 0.2, 1],
                  }}
                />
              </g>
            )}
          </motion.g>

          {/* Energy pulse running up one edge and down the other */}
          {play && (
            <motion.path
              d="M81 184 L93 48 L124.5 11.5 L156.5 48 L168 184"
              fill="none"
              stroke="#f8fdb0"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeDasharray="22 330"
              filter={url("glow")}
              initial={{ strokeDashoffset: 0, opacity: 0 }}
              animate={{ strokeDashoffset: [0, -352], opacity: [0, 1, 1, 0] }}
              transition={{
                duration: 3.4,
                delay: 1.6,
                repeat: Infinity,
                repeatDelay: 1.4,
                ease: "linear",
              }}
            />
          )}
        </motion.svg>
      </motion.div>
    </motion.div>
  );
}
