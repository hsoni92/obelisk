"use client";

import { BlueprintGate } from "@/components/blueprint/gate";
import { SyncBadge } from "@/components/blueprint/sync-badge";
import { UnlockBanner } from "@/components/blueprint/unlock-banner";
import { AppShell } from "@/components/layout/app-shell";
import { Badge } from "@/components/ui/badge";
import { statusPollOptions, useGetStatusQuery } from "@/lib/api";
import { SERVICES_SURFACE } from "@/lib/filters";
import { navIdFromPathname } from "@/lib/nav";
import { useSurfaceFilters } from "@/lib/store/hooks";
import { usePathname, useRouter } from "next/navigation";

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

  const { data: status } = useGetStatusQuery(undefined, statusPollOptions);

  const meta = (
    <>
      <SyncBadge />
      {status?.profile && <Badge variant="info">{status.profile}</Badge>}
      {status?.reconcileMode === "off" && (
        <Badge variant="warning">reconcile off</Badge>
      )}
    </>
  );

  const onSearchChange =
    active === "services"
      ? (value: string) => {
          setServiceQuery(value);
          const path = "/services";
          const url = value.trim()
            ? `${path}?q=${encodeURIComponent(value)}`
            : path;
          router.replace(url, { scroll: false });
        }
      : undefined;

  return (
    <AppShell
      active={active}
      meta={meta}
      searchQuery={active === "services" ? serviceFilters.query : ""}
      onSearchChange={onSearchChange}
    >
      <UnlockBanner />
      {children}
    </AppShell>
  );
}