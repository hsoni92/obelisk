export type NavId = "overview" | "services" | "dr" | "exposure" | "platform";

export const NAV_ROUTES: Record<NavId, string> = {
  overview: "/overview",
  services: "/services",
  dr: "/dr",
  exposure: "/exposure",
  platform: "/platform",
};

export const PAGE_TITLES: Record<
  NavId,
  { title: string; subtitle: string }
> = {
  overview: {
    title: "Overview",
    subtitle:
      "Platform and data services health from k3s/Helm, with recent state history.",
  },
  services: {
    title: "Services",
    subtitle:
      "Manage instances in one table — sort, filter, copy connection strings per row.",
  },
  dr: {
    title: "Disaster recovery",
    subtitle:
      "Configure R2/S3, test upload access, then run backups for every running data service.",
  },
  exposure: {
    title: "Exposure",
    subtitle: "Kong HTTP routes and optional Tailscale Funnel.",
  },
  platform: {
    title: "Platform",
    subtitle: "Console settings and host desired state.",
  },
};

const PATH_TO_NAV: Record<string, NavId> = Object.fromEntries(
  (Object.entries(NAV_ROUTES) as [NavId, string][]).map(([id, path]) => [
    path,
    id,
  ]),
) as Record<string, NavId>;

export function navIdFromPathname(pathname: string): NavId | null {
  return PATH_TO_NAV[pathname] ?? null;
}