import type { ConnectionsResponse, ServiceInstance } from "@/lib/api";

export type InstanceConnection =
  NonNullable<ConnectionsResponse["instances"]>[number];

export type ServiceRow = {
  rowKey: string;
  instance: ServiceInstance;
  state: string;
  connection?: InstanceConnection;
};

export function serviceRowKey(inst: { type: string; id: string }) {
  return `${inst.type}-${inst.id}`;
}

export function buildServiceRows(
  instances: ServiceInstance[],
  serviceStates: Record<string, string> | undefined,
  connections: ConnectionsResponse | null,
  statusLoaded: boolean,
): ServiceRow[] {
  const connMap = new Map<string, InstanceConnection>();
  for (const c of connections?.instances ?? []) {
    connMap.set(serviceRowKey(c), c);
  }

  return instances.map((inst) => {
    const key = serviceRowKey(inst);
    const state =
      serviceStates?.[`${inst.type}/${inst.id}`] ??
      inst.state ??
      (statusLoaded ? "stopped" : "loading");
    return {
      rowKey: key,
      instance: inst,
      state,
      connection: connMap.get(key),
    };
  });
}

export type ConnectionEntry = { label: string; value: string };

export function connectionEntries(
  conn?: InstanceConnection,
): ConnectionEntry[] {
  if (!conn) return [];
  const out: ConnectionEntry[] = [];
  if (conn.postgres) out.push({ label: "PostgreSQL", value: conn.postgres });
  if (conn.mysql) out.push({ label: "MySQL", value: conn.mysql });
  if (conn.redis) out.push({ label: "Redis", value: conn.redis });
  if (conn.s3_http) out.push({ label: "S3 HTTP", value: conn.s3_http });
  if (conn.qdrant_http) out.push({ label: "Qdrant REST", value: conn.qdrant_http });
  if (conn.qdrant_grpc) out.push({ label: "Qdrant gRPC", value: conn.qdrant_grpc });
  if (conn.searxng_http) out.push({ label: "SearXNG", value: conn.searxng_http });
  return out;
}

export function connectionPreview(conn?: InstanceConnection): string | null {
  const entries = connectionEntries(conn);
  return entries[0]?.value ?? null;
}

export function stateBadgeVariant(state: string) {
  if (state === "running") return "success" as const;
  if (state === "pending") return "warning" as const;
  if (state === "degraded") return "live" as const;
  return "default" as const;
}