"use client";

import { ObeliskWordmark } from "@/components/brand/obelisk-logo";
import { NAV_ROUTES, type NavId } from "@/lib/nav";
import { cn } from "@/lib/utils";
import {
  Gauge,
  HardDriveDownload,
  LayoutDashboard,
  Menu,
  Search,
  Settings,
  Share2,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import * as React from "react";
import { Button } from "@/components/ui/button";

const NAV: { id: NavId; label: string; icon: LucideIcon }[] = [
  { id: "overview", label: "Overview", icon: Gauge },
  { id: "services", label: "Services", icon: LayoutDashboard },
  { id: "dr", label: "Disaster Recovery", icon: HardDriveDownload },
  { id: "exposure", label: "Exposure", icon: Share2 },
  { id: "platform", label: "Platform", icon: Settings },
];

export type { NavId };

type AppShellProps = {
  active: NavId;
  children: React.ReactNode;
  meta?: React.ReactNode;
  searchQuery?: string;
  onSearchChange?: (value: string) => void;
};

export function AppShell({
  active,
  children,
  meta,
  searchQuery = "",
  onSearchChange,
}: AppShellProps) {
  const [rail, setRail] = React.useState(false);

  return (
    <div className="min-h-screen text-foreground flex">
      <aside
        className={cn(
          "sticky top-0 h-screen shrink-0 py-5 pl-5 transition-[width] flex flex-col",
          rail ? "w-[76px]" : "w-60",
        )}
      >
        <div className={cn("px-1", rail && "px-0")}>
          <ObeliskWordmark compact={rail} />
        </div>

        <nav className="mt-6 flex flex-col gap-1 rounded-[28px] bg-card p-2 shadow-rb-2">
          {NAV.map((item) => {
            const selected = active === item.id;
            const Icon = item.icon;
            return (
              <Link
                key={item.id}
                href={NAV_ROUTES[item.id]}
                title={item.label}
                className={cn(
                  "flex items-center gap-2.5 rounded-full py-2.5 pl-4 pr-4 text-left text-sm transition-colors",
                  selected
                    ? "bg-ink font-medium text-ink-foreground"
                    : "font-normal text-muted-foreground hover:bg-muted hover:text-foreground",
                  rail && "justify-center px-0",
                )}
              >
                <Icon className="size-4 shrink-0" />
                {!rail && item.label}
              </Link>
            );
          })}
        </nav>

        {!rail && (
          <div className="mt-auto rounded-[28px] bg-feature p-4 text-feature-foreground shadow-rb-2">
            <p className="text-sm font-medium">Middleware appliance</p>
            <p className="mt-1 text-xs leading-4 text-white/70">
              Platform first, data services on demand.
            </p>
          </div>
        )}
      </aside>

      <div className="flex-1 min-w-0 flex flex-col">
        <header className="sticky top-0 z-30 flex h-16 items-center gap-3 px-6 backdrop-blur-sm">
          <Button
            type="button"
            variant="icon"
            size="icon"
            aria-label="Toggle sidebar"
            onClick={() => setRail((r) => !r)}
          >
            <Menu className="size-4" />
          </Button>
          <div className="flex-1 max-w-md">
            <label className="sr-only" htmlFor="admin-search">
              Search
            </label>
            <div className="flex h-10 items-center gap-2 rounded-full bg-card px-4 text-muted-foreground shadow-rb-1 focus-within:ring-2 focus-within:ring-ring/60">
              <Search className="size-3.5 shrink-0" />
              <input
                id="admin-search"
                placeholder={
                  active === "services"
                    ? "Filter services by name, type, endpoint…"
                    : "Search admin…"
                }
                value={searchQuery}
                onChange={(e) => onSearchChange?.(e.target.value)}
                className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none"
              />
            </div>
          </div>
          <div className="ml-auto flex items-center gap-2">{meta}</div>
        </header>

        <main className="flex-1 min-w-0 px-6 pb-10 pt-4 max-w-[1400px] w-full mx-auto">
          {children}
        </main>
      </div>
    </div>
  );
}
