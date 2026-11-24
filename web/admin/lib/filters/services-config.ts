import {
  connectionPreview,
  type ServiceRow,
} from "@/lib/service-rows";
import type { FilterConfig } from "./engine";
import type { SurfaceFilterState } from "./types";

export const SERVICES_SURFACE = "services" as const;

export const SERVICE_TYPES = [
  "postgres",
  "mysql",
  "mongodb",
  "redis",
  "rustfs",
  "qdrant",
  "searxng",
  "redpanda",
  "unimeter",
  "bunqueue",
  "browserless",
] as const;

export const SERVICE_STATES = [
  "running",
  "stopped",
  "starting",
  "stopping",
  "pending",
  "loading",
  "degraded",
] as const;

export const SERVICE_TYPE_LABEL: Record<(typeof SERVICE_TYPES)[number], string> =
  {
    postgres: "PostgreSQL",
    mysql: "MySQL",
    mongodb: "MongoDB",
    redis: "Redis",
    rustfs: "RustFS",
    qdrant: "Qdrant",
    searxng: "SearXNG",
    redpanda: "Redpanda",
    unimeter: "Unimeter",
    bunqueue: "bunqueue",
    browserless: "Browserless",
  };

export function defaultServicesFilters(): SurfaceFilterState {
  return {
    query: "",
    sets: {
      types: [],
      states: [],
    },
    tri: {
      enabled: "all",
      endpoint: "all",
    },
  };
}

export const servicesFilterConfig: FilterConfig<ServiceRow> = {
  text: {
    getHaystack: (row) => {
      const inst = row.instance;
      const endpoint = connectionPreview(row.connection) ?? "";
      return [
        inst.id,
        inst.type,
        inst.display_name ?? "",
        row.state,
        endpoint,
        inst.enabled ? "enabled" : "disabled",
      ].join(" ");
    },
  },
  sets: [
    {
      key: "types",
      getValues: (row) => row.instance.type,
    },
    {
      key: "states",
      getValues: (row) => row.state,
    },
  ],
  tris: [
    {
      key: "enabled",
      test: (row) => row.instance.enabled,
    },
    {
      key: "endpoint",
      test: (row) => {
        if (row.state !== "running") return false;
        return connectionPreview(row.connection) != null;
      },
    },
  ],
};