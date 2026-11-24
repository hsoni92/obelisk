"use client";

import { useSafeReducedMotion } from "@/lib/use-reduced-motion";
import { AnimatedObeliskMark } from "@/components/brand/animated-obelisk";
import { cn } from "@/lib/utils";

/** Scene coordinates (viewBox 0 0 640 640). The monolith stands on (CX, CY). */
const CX = 320;
const CY = 338;
const VP_Y = 236; // vanishing point of the floor grid

const RINGS = [
  { id: "ring-a", rx: 250, ry: 80 },
  { id: "ring-b", rx: 196, ry: 62 },
  { id: "ring-c", rx: 142, ry: 44 },
] as const;

/** Services that orbit the appliance. `phase` offsets where each one starts. */
const NODES = [
  { label: "PostgreSQL", ring: 0, dur: 34, phase: 0.0 },
  { label: "Kong", ring: 0, dur: 34, phase: 0.52 },
  { label: "Redis", ring: 1, dur: 27, phase: 0.18 },
  { label: "Tailscale", ring: 1, dur: 27, phase: 0.68 },
  { label: "Qdrant", ring: 2, dur: 21, phase: 0.36 },
  { label: "RustFS", ring: 2, dur: 21, phase: 0.86 },
] as const;

/** Packets flowing in from the edges toward the base. */
const FLOWS = [
  { d: "M18 560 C150 548 252 480 312 360", dur: 5.6, begin: 0 },
  { d: "M622 552 C496 540 392 480 328 360", dur: 6.2, begin: -2.4 },
  { d: "M30 238 C124 286 226 330 306 352", dur: 5.9, begin: -1.1 },
  { d: "M612 246 C520 292 420 332 334 352", dur: 6.6, begin: -3.7 },
] as const;

function ellipsePath(rx: number, ry: number) {
  return `M${CX - rx} ${CY} A${rx} ${ry} 0 1 1 ${CX + rx} ${CY} A${rx} ${ry} 0 1 1 ${CX - rx} ${CY}`;
}

/** Lower (front) half of an ellipse, left to right. */
function frontArc(rx: number, ry: number) {
  return `M${CX - rx} ${CY} A${rx} ${ry} 0 0 0 ${CX + rx} ${CY}`;
}

function pointOnRing(ring: number, t: number) {
  const { rx, ry } = RINGS[ring];
  const a = Math.PI + t * Math.PI * 2; // start on the left, clockwise on screen
  return { x: CX + rx * Math.cos(a), y: CY + ry * Math.sin(a) };
}

/** Deterministic pseudo-random in [0, 1) so server and client markup match. */
function rand(seed: number) {
  const x = Math.sin(seed * 127.1 + 311.7) * 43758.5453;
  return x - Math.floor(x);
}

const GRID_V = Array.from({ length: 19 }, (_, i) => -380 + i * 80);
const GRID_H = Array.from({ length: 11 }, (_, i) => CY + Math.pow((i + 1) / 11, 1.75) * (640 - CY));

/** Two decimals: long floats serialize differently on server and client and break hydration. */
const r2 = (n: number) => Math.round(n * 100) / 100;

const PARTICLES = Array.from({ length: 18 }, (_, i) => ({
  left: r2(6 + rand(i + 1) * 88),
  bottom: r2(4 + rand(i + 40) * 34),
  size: r2(2 + rand(i + 80) * 2.6),
  dur: r2(8 + rand(i + 120) * 8),
  delay: r2(-rand(i + 160) * 14),
  rise: Math.round(150 + rand(i + 200) * 190),
}));

/**
 * Decorative hero: a monolith on a perspective floor with service nodes
 * orbiting it and data packets flowing in. Pure SVG + CSS transforms, so it
 * stays cheap. Reduced-motion users get the same picture, still.
 */
export function ObeliskScene({ className }: { className?: string }) {
  const reduce = useSafeReducedMotion();
  const play = !reduce;

  return (
    <div
      aria-hidden
      className={cn("relative mx-auto aspect-square w-full max-w-[640px]", className)}
    >
      {/* ---- Back layer: floor, rings, nodes, flows ---- */}
      <svg viewBox="0 0 640 640" className="absolute inset-0 size-full overflow-visible">
        <defs>
          <radialGradient id="os-fade" cx={CX} cy={CY} r="330" gradientUnits="userSpaceOnUse">
            <stop offset="0" stopColor="#fff" stopOpacity="1" />
            <stop offset="0.65" stopColor="#fff" stopOpacity="0.55" />
            <stop offset="1" stopColor="#fff" stopOpacity="0" />
          </radialGradient>
          <mask id="os-floor-mask">
            <rect x="-120" y="0" width="880" height="640" fill="url(#os-fade)" />
          </mask>
          <linearGradient id="os-beam" x1="0" y1="1" x2="0" y2="0">
            <stop offset="0" stopColor="#d7f249" stopOpacity="0.26" />
            <stop offset="0.7" stopColor="#d7f249" stopOpacity="0.07" />
            <stop offset="1" stopColor="#d7f249" stopOpacity="0" />
          </linearGradient>
          <linearGradient id="os-beam-line" x1="0" y1="1" x2="0" y2="0">
            <stop offset="0" stopColor="#f2fc9a" stopOpacity="0.9" />
            <stop offset="1" stopColor="#f2fc9a" stopOpacity="0" />
          </linearGradient>
          <radialGradient id="os-pool" cx={CX} cy={CY + 10} r="260" gradientUnits="userSpaceOnUse">
            <stop offset="0" stopColor="#d7f249" stopOpacity="0.22" />
            <stop offset="0.55" stopColor="#d7f249" stopOpacity="0.06" />
            <stop offset="1" stopColor="#d7f249" stopOpacity="0" />
          </radialGradient>
          {RINGS.map((r) => (
            <path key={r.id} id={`os-${r.id}`} d={ellipsePath(r.rx, r.ry)} fill="none" />
          ))}
          {FLOWS.map((f, i) => (
            <path key={i} id={`os-flow-${i}`} d={f.d} fill="none" />
          ))}
        </defs>

        {/* Light pool on the floor */}
        <ellipse cx={CX} cy={CY + 10} rx="300" ry="120" fill="url(#os-pool)" />

        {/* Perspective floor */}
        <g mask="url(#os-floor-mask)" stroke="#d7f249" strokeOpacity="0.2" strokeWidth="1">
          {GRID_H.map((y) => (
            <line key={y} x1="-120" x2="760" y1={y} y2={y} />
          ))}
          {GRID_V.map((bx) => {
            const k = (CY - VP_Y) / (640 - VP_Y);
            return (
              <line key={bx} x1={CX + (bx - CX) * k} y1={CY} x2={bx} y2="640" />
            );
          })}
        </g>

        {/* Beam from the apex */}
        <path d={`M${CX} 134 L244 -30 L396 -30 Z`} fill="url(#os-beam)" />
        <rect x={CX - 0.75} y="-30" width="1.5" height="164" fill="url(#os-beam-line)" />

        {/* Orbit rings */}
        {RINGS.map((r, i) => (
          <use
            key={r.id}
            href={`#os-${r.id}`}
            stroke="#fff"
            strokeOpacity={0.16 - i * 0.025}
            strokeWidth="1"
            strokeDasharray={i === 1 ? "2 7" : undefined}
          />
        ))}

        {/* Data flowing in */}
        {FLOWS.map((f, i) => (
          <g key={i}>
            <use
              href={`#os-flow-${i}`}
              stroke="#d7f249"
              strokeOpacity="0.28"
              strokeWidth="1"
              strokeDasharray="3 7"
              className={play ? "auth-dash" : undefined}
            />
            {play && (
              <g>
                <animateMotion dur={`${f.dur}s`} begin={`${f.begin}s`} repeatCount="indefinite">
                  <mpath href={`#os-flow-${i}`} />
                </animateMotion>
                <circle r="7" fill="#d7f249" opacity="0.18" />
                <circle r="2.6" fill="#f2fc9a" />
              </g>
            )}
          </g>
        ))}

        {/* Orbiting services */}
        {NODES.map((n) => {
          const body = (
            <>
              <circle r="9" fill="#d7f249" opacity="0.14" />
              <circle r="3.6" fill="#d7f249" />
              <circle r="9" fill="none" stroke="#d7f249" strokeOpacity="0.5" strokeWidth="0.8" />
              <text
                x="14"
                y="3.2"
                fontSize="9.5"
                fill="#fff"
                fillOpacity="0.62"
                style={{ fontFamily: "var(--font-roboto-mono), monospace", letterSpacing: "0.04em" }}
              >
                {n.label}
              </text>
            </>
          );
          if (!play) {
            const p = pointOnRing(n.ring, n.phase);
            return (
              <g key={n.label} transform={`translate(${p.x} ${p.y})`}>
                {body}
              </g>
            );
          }
          return (
            <g key={n.label}>
              <animateMotion
                dur={`${n.dur}s`}
                begin={`${-n.phase * n.dur}s`}
                repeatCount="indefinite"
                rotate="0"
              >
                <mpath href={`#os-${RINGS[n.ring].id}`} />
              </animateMotion>
              {/* Dim on the far side of the orbit (t = 0.25) and brighten in front (t = 0.75). */}
              <animate
                attributeName="opacity"
                dur={`${n.dur}s`}
                begin={`${-n.phase * n.dur}s`}
                repeatCount="indefinite"
                values="0.85;0.3;0.85;1;0.85"
                keyTimes="0;0.25;0.5;0.75;1"
              />
              {body}
            </g>
          );
        })}
      </svg>

      {/* ---- The monolith ---- */}
      <div className="absolute" style={{ left: "32%", top: "17.2%", width: "36%" }}>
        <AnimatedObeliskMark decorative />
      </div>

      {/* ---- Front layer: foreground arcs ---- */}
      <svg viewBox="0 0 640 640" className="pointer-events-none absolute inset-0 size-full overflow-visible">
        <defs>
          <linearGradient id="os-front" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0" stopColor="#d7f249" stopOpacity="0" />
            <stop offset="0.5" stopColor="#d7f249" stopOpacity="0.55" />
            <stop offset="1" stopColor="#d7f249" stopOpacity="0" />
          </linearGradient>
        </defs>
        {RINGS.map((r) => (
          <path
            key={r.id}
            d={frontArc(r.rx, r.ry)}
            fill="none"
            stroke="url(#os-front)"
            strokeWidth="1.2"
          />
        ))}

      </svg>

      {/* ---- Rising particles ---- */}
      {play &&
        PARTICLES.map((p, i) => (
          <span
            key={i}
            className="auth-particle absolute rounded-full bg-[#f2fc9a]"
            style={
              {
                left: `${p.left}%`,
                bottom: `${p.bottom}%`,
                width: p.size,
                height: p.size,
                animationDuration: `${p.dur}s`,
                animationDelay: `${p.delay}s`,
                "--rise": `${p.rise}px`,
              } as React.CSSProperties
            }
          />
        ))}
    </div>
  );
}
