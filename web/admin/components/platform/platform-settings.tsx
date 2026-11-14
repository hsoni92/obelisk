"use client";

import { cn } from "@/lib/utils";
import {
  readTheme,
  setTheme,
  subscribeTheme,
  systemPrefersDark,
  type ThemeMode,
} from "@/lib/theme";
import { Monitor, Moon, Palette, Sun, type LucideIcon } from "lucide-react";
import { useSyncExternalStore } from "react";

const MODES: {
  id: ThemeMode;
  label: string;
  hint: string;
  icon: LucideIcon;
}[] = [
  { id: "light", label: "Light", hint: "Warm paper background", icon: Sun },
  { id: "dark", label: "Dark", hint: "Night palette", icon: Moon },
  { id: "system", label: "System", hint: "Match this device", icon: Monitor },
];

function subscribeSystem(onChange: () => void) {
  const media = window.matchMedia("(prefers-color-scheme: dark)");
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

export function PlatformSettings() {
  const theme = useSyncExternalStore(subscribeTheme, readTheme, () => "light" as const);
  const systemDark = useSyncExternalStore(
    subscribeSystem,
    systemPrefersDark,
    () => false,
  );
  const resolved = theme === "system" ? (systemDark ? "dark" : "light") : theme;

  return (
    <div className="grid max-w-3xl gap-6 md:grid-cols-[11rem_minmax(0,1fr)]">
      <nav aria-label="Platform settings">
        <p className="mb-2 px-3 text-eyebrow text-muted-foreground">Settings</p>
        <button
          type="button"
          aria-current="page"
          className="flex w-full items-center gap-2 rounded-lg border-l-2 border-broadcast bg-card px-3 py-2 text-left text-[13px] font-semibold text-broadcast"
        >
          <Palette className="size-4 shrink-0" />
          Appearance
        </button>
      </nav>

      <section aria-labelledby="appearance-heading">
        <h2 id="appearance-heading" className="text-page-title">
          Appearance
        </h2>
        <p className="mt-0.5 text-body text-muted-foreground">
          Theme for this browser. It stays on this machine and does not change
          the appliance.
        </p>

        <div className="mt-4 rounded-xl border border-border bg-card p-4 shadow-rb-2">
          <p className="text-section-title">Theme</p>
          <p className="mt-0.5 text-metadata text-muted-foreground">
            {theme === "dark"
              ? "Dark theme is on."
              : theme === "light"
                ? "Light theme is on."
                : `Following this device (${resolved}).`}
          </p>

          <div className="mt-4 grid grid-cols-3 gap-3">
            {MODES.map((mode) => {
              const selected = theme === mode.id;
              const Icon = mode.icon;
              return (
                <button
                  key={mode.id}
                  type="button"
                  aria-pressed={selected}
                  onClick={() => setTheme(mode.id)}
                  className={cn(
                    "rounded-xl border bg-background p-2 text-left transition-colors hover:bg-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                    selected
                      ? "border-broadcast ring-1 ring-broadcast"
                      : "border-border",
                  )}
                >
                  <ThemePreview mode={mode.id} />
                  <span className="mt-2 flex items-center gap-1.5 px-0.5 text-[13px] font-medium text-foreground">
                    <Icon className="size-3.5 text-muted-foreground" />
                    {mode.label}
                  </span>
                  <span className="mt-0.5 block px-0.5 text-[11px] leading-4 text-muted-foreground">
                    {mode.hint}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <p className="mt-6 text-metadata text-muted-foreground">
          Desired-state editor for obelisk.yaml — coming next.
        </p>
      </section>
    </div>
  );
}

function ThemePreview({ mode }: { mode: ThemeMode }) {
  if (mode === "system") {
    return (
      <span className="flex h-14 overflow-hidden rounded-lg border border-black/10">
        <span className="flex-1 bg-[#faf8f8]" />
        <span className="flex-1 bg-[#141010]" />
      </span>
    );
  }
  const dark = mode === "dark";
  return (
    <span
      className={cn(
        "flex h-14 flex-col justify-center gap-1.5 rounded-lg border px-2.5",
        dark
          ? "border-white/10 bg-[#141010]"
          : "border-[#eee9e9] bg-[#faf8f8]",
      )}
    >
      <span
        className={cn(
          "h-1.5 w-8 rounded-full",
          dark ? "bg-[#f4eeee]" : "bg-[#161212]",
        )}
      />
      <span
        className={cn(
          "h-1.5 w-12 rounded-full",
          dark ? "bg-white/25" : "bg-black/15",
        )}
      />
      <span className="h-1.5 w-5 rounded-full bg-[#bb0000]" />
    </span>
  );
}
