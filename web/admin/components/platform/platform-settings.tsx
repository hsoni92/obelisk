"use client";

import { BlueprintPanel } from "@/components/blueprint/blueprint-panel";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import {
  readTheme,
  setTheme,
  subscribeTheme,
  systemPrefersDark,
  type ThemeMode,
} from "@/lib/theme";
import { GitBranch, Monitor, Moon, Palette, Sun, type LucideIcon } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState, useSyncExternalStore } from "react";

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

const PLATFORM_TABS = [
  { id: "appearance", label: "Appearance", icon: Palette },
  { id: "blueprint", label: "Blueprint", icon: GitBranch },
] as const;

type PlatformTab = (typeof PLATFORM_TABS)[number]["id"];

function isPlatformTab(value: string): value is PlatformTab {
  return PLATFORM_TABS.some((tab) => tab.id === value);
}

function noopSubscribe() {
  return () => {};
}

function subscribeSystem(onChange: () => void) {
  const media = window.matchMedia("(prefers-color-scheme: dark)");
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

const tabTriggerClass =
  "group flex w-full outline-none focus:outline-none focus-visible:ring-0 focus-visible:outline-none items-center justify-start gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] font-medium text-muted-foreground shadow-none transition-colors hover:bg-card/70 hover:text-foreground data-[state=active]:bg-card data-[state=active]:text-foreground data-[state=active]:shadow-rb-2";

export function PlatformSettings() {
  const router = useRouter();
  const [tab, setTab] = useState<PlatformTab>("appearance");
  // False during server render and hydration, true once the client can read the
  // hash. Keeps the default tab from flashing before the linked one.
  const ready = useSyncExternalStore(
    noopSubscribe,
    () => true,
    () => false,
  );
  const theme = useSyncExternalStore(subscribeTheme, readTheme, () => "light" as const);
  const systemDark = useSyncExternalStore(
    subscribeSystem,
    systemPrefersDark,
    () => false,
  );
  const resolved = theme === "system" ? (systemDark ? "dark" : "light") : theme;

  useEffect(() => {
    function syncFromHash() {
      const hash = window.location.hash.replace(/^#/, "");
      // Tailscale and gateway moved to Connectivity. Keep old links working.
      if (hash === "tailscale") {
        router.replace("/connectivity");
        return;
      }
      if (isPlatformTab(hash)) setTab(hash);
    }
    syncFromHash();
    window.addEventListener("hashchange", syncFromHash);
    return () => window.removeEventListener("hashchange", syncFromHash);
  }, [router]);

  function onTabChange(value: string) {
    if (!isPlatformTab(value)) return;
    setTab(value);
    const path = `${window.location.pathname}#${value}`;
    window.history.replaceState(null, "", path);
  }

  return (
    <Tabs
      value={tab}
      onValueChange={onTabChange}
      className={`max-w-3xl ${ready ? "" : "invisible"}`}
    >
      <div className="grid gap-6 md:grid-cols-[13rem_minmax(0,1fr)] md:gap-8">
        <TabsList
          aria-label="Platform settings"
          className="h-auto flex-row gap-1 rounded-none bg-transparent p-0 shadow-none md:sticky md:top-4 md:flex-col md:items-stretch md:self-start"
        >
                    {PLATFORM_TABS.map((item) => {
            const Icon = item.icon;
            return (
              <TabsTrigger key={item.id} value={item.id} className={tabTriggerClass}>
                <Icon className="size-4 shrink-0 opacity-70 group-data-[state=active]:text-broadcast group-data-[state=active]:opacity-100" />
                {item.label}
              </TabsTrigger>
            );
          })}
        </TabsList>

        <div className="min-w-0">
          <TabsContent value="appearance" className="mt-0">
            <section aria-labelledby="appearance-heading">
              <h2 id="appearance-heading" className="text-page-title">
                Appearance
              </h2>
              <p className="mt-0.5 text-body text-muted-foreground">
                Theme for this browser. It stays on this device and does not change the appliance.
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
            </section>
          </TabsContent>

          <TabsContent value="blueprint" className="mt-0">
            <BlueprintPanel />
          </TabsContent>
        </div>
      </div>
    </Tabs>
  );
}

function ThemePreview({ mode }: { mode: ThemeMode }) {
  if (mode === "system") {
    return (
      <span className="flex h-14 overflow-hidden rounded-xl border border-black/10">
        <span className="flex-1 bg-[#ebebed]" />
        <span className="flex-1 bg-[#101412]" />
      </span>
    );
  }
  const dark = mode === "dark";
  return (
    <span
      className={cn(
        "flex h-14 flex-col justify-center gap-1.5 rounded-xl border px-2.5",
        dark ? "border-white/10 bg-[#101412]" : "border-[#e9e9ed] bg-[#ebebed]",
      )}
    >
      <span
        className={cn("h-1.5 w-8 rounded-full", dark ? "bg-[#f1f3ee]" : "bg-[#0e0f0c]")}
      />
      <span
        className={cn("h-1.5 w-12 rounded-full", dark ? "bg-white/25" : "bg-black/15")}
      />
      <span className="h-1.5 w-5 rounded-full bg-[#d7f249]" />
    </span>
  );
}