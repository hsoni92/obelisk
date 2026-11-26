"use client";

import { ObeliskWordmark } from "@/components/brand/obelisk-logo";
import { NAV_ROUTES, type NavId } from "@/lib/nav";
import { cn } from "@/lib/utils";
import {
  Gauge,
  HardDriveDownload,
  FolderKanban,
  LayoutDashboard,
  Menu,
  Network,
  Search,
  Settings,
  Share2,
  type LucideIcon,
} from "lucide-react";
import { SidebarFooter } from "@/components/layout/sidebar-footer";
import { MotionConfig, motion } from "framer-motion";
import Link from "next/link";
import * as React from "react";
import { Button } from "@/components/ui/button";

const NAV: { id: NavId; label: string; icon: LucideIcon }[] = [
  { id: "overview", label: "Overview", icon: Gauge },
  { id: "projects", label: "Projects", icon: FolderKanban },
  { id: "services", label: "Services", icon: LayoutDashboard },
  { id: "dr", label: "Disaster Recovery", icon: HardDriveDownload },
  { id: "exposure", label: "Exposure", icon: Share2 },
  { id: "connectivity", label: "Connectivity", icon: Network },
  { id: "platform", label: "Platform", icon: Settings },
];

export type { NavId };

type AppShellProps = {
  active: NavId;
  children: React.ReactNode;
  meta?: React.ReactNode;
  searchQuery?: string;
  onSearchChange?: (value: string) => void;
  defaultRail?: boolean;
};

export function AppShell({
  active,
  children,
  meta,
  searchQuery = "",
  onSearchChange,
  defaultRail = false,
}: AppShellProps) {
  const [rail, setRail] = React.useState(defaultRail);

  const ease = [0.32, 0.72, 0, 1] as const;

  return (
    <MotionConfig reducedMotion="user">
      <div className="flex h-dvh overflow-hidden text-foreground">
        <motion.aside
          initial={false}
          animate={{ width: rail ? 76 : 240 }}
          // Collapse: labels fade first, then the rail narrows. Expand: width first.
          transition={{ duration: 0.26, ease, delay: rail ? 0.1 : 0 }}
          className="flex h-full shrink-0 flex-col overflow-clip py-5 pl-5"
        >
          <div className="pl-1">
            <ObeliskWordmark compact={rail} />
          </div>

          <>
            <nav className="mt-6 w-full flex flex-col gap-1 rounded-[28px] bg-card p-2 shadow-rb-2">
              {NAV.map((item) => {
                const selected = active === item.id;
                const Icon = item.icon;
                return (
                  <Link
                    key={item.id}
                    href={NAV_ROUTES[item.id]}
                    title={item.label}
                    aria-label={item.label}
                    className={cn(
                      "relative flex w-full shrink-0 items-center overflow-clip rounded-full py-2.5 pl-3 pr-3 text-left text-sm transition-colors",
                      selected
                        ? "font-medium text-ink-foreground"
                        : "font-normal text-muted-foreground hover:bg-muted hover:text-foreground",
                    )}
                  >
                    <span
                      aria-hidden
                      className={cn(
                        "absolute inset-0 rounded-full bg-ink transition-opacity duration-200",
                        selected ? "opacity-100" : "opacity-0",
                      )}
                    />
                    <Icon className="relative size-4 shrink-0" />
                    <motion.span
                      initial={false}
                      animate={{
                        opacity: rail ? 0 : 1,
                        width: rail ? 0 : "auto",
                        marginLeft: rail ? 0 : 10,
                      }}
                      transition={{
                        opacity: { duration: 0.14, delay: rail ? 0 : 0.16 },
                        width: { duration: 0.2, delay: rail ? 0.1 : 0 },
                        marginLeft: { duration: 0.2, delay: rail ? 0.1 : 0 },
                      }}
                      aria-hidden={rail}
                      className="relative overflow-clip whitespace-nowrap"
                    >
                      {item.label}
                    </motion.span>
                  </Link>
                );
              })}
            </nav>
          </>

          <SidebarFooter rail={rail} />
        </motion.aside>

        <div className="scroll-stable flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto overscroll-y-contain">
          <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center gap-3 px-6 backdrop-blur-sm">
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

          <motion.main
            key={active}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, ease }}
            className="flex-1 min-w-0 px-6 pb-10 pt-4 max-w-[1400px] w-full mx-auto"
          >
            {children}
          </motion.main>
        </div>
      </div>
    </MotionConfig>
  );
}
