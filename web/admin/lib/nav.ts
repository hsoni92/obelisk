export type NavId =
  | "overview"
  | "services"
  | "projects"
  | "dr"
  | "exposure"
  | "platform";

export const NAV_ROUTES: Record<NavId, string> = {
  overview: "/overview",
  services: "/services",
  projects: "/projects",
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
  projects: {
    title: "Projects",
    subtitle:
      "Group data services and exposure gateways for connection discovery URLs.",
  },
  dr: {
    title: "Disaster Recovery",
    subtitle:
      "Add S3/R2 destinations, set global policy and replication, then back up every running data service.",
  },
  exposure: {
    title: "Exposure",
    subtitle: "Kong HTTP routes and optional Tailscale Funnel.",
  },
  platform: {
    title: "Platform",
    subtitle:
      "Tailscale, gateway role, appearance, and blueprint — always available here.",
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