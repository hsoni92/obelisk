"use client";

import { BlueprintGate } from "@/components/blueprint/gate";
import { UnlockBanner } from "@/components/blueprint/unlock-banner";
import { RecentActivityDrawer } from "@/components/activity/recent-activity-drawer";
import { AppShell } from "@/components/layout/app-shell";
import { Badge } from "@/components/ui/badge";
import { statusPollOptions, useGetStatusQuery } from "@/lib/api";
import { SERVICES_SURFACE } from "@/lib/filters";
import { navIdFromPathname } from "@/lib/nav";
import { useSurfaceFilters } from "@/lib/store/hooks";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const SEARCH_DEBOUNCE_MS = 300;

export function ConsoleLayout({ children }: { children: React.ReactNode }) {
  return (
    <BlueprintGate>
      <ConsoleBody>{children}</ConsoleBody>
    </BlueprintGate>
  );
}

function ConsoleBody({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const active = navIdFromPathname(pathname) ?? "overview";
  const { filters: serviceFilters, setQuery: setServiceQuery } =
    useSurfaceFilters(SERVICES_SURFACE);
  const [searchDraft, setSearchDraft] = useState(serviceFilters.query);

  useEffect(() => {
    if (active === "services") {
      setSearchDraft(serviceFilters.query);
    }
  }, [active, serviceFilters.query]);

  useEffect(() => {
    if (active !== "services") return;
    const handle = window.setTimeout(() => {
      const value = searchDraft;
      if (value === serviceFilters.query) return;
      setServiceQuery(value);
      const path = "/services";
      const url = value.trim()
        ? `${path}?q=${encodeURIComponent(value)}`
        : path;
      router.replace(url, { scroll: false });
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(handle);
  }, [
    active,
    router,
    searchDraft,
    serviceFilters.query,
    setServiceQuery,
  ]);

  const { data: status } = useGetStatusQuery(undefined, statusPollOptions);

  const meta = (
    <>
      <RecentActivityDrawer />
      {status?.reconcileMode === "off" && (
        <Badge variant="warning">reconcile off</Badge>
      )}
    </>
  );

  const onSearchChange =
    active === "services" ? (value: string) => setSearchDraft(value) : undefined;

  return (
    <AppShell
      active={active}
      meta={meta}
      searchQuery={active === "services" ? searchDraft : ""}
      onSearchChange={onSearchChange}
    >
      <UnlockBanner />
      {children}
    </AppShell>
  );
}
