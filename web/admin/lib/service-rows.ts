import type { ConnectionsResponse, ServiceInstance } from "@/lib/api";
import type { VitalsRow } from "@/lib/types";

export type InstanceConnection = NonNullable<
  ConnectionsResponse["instances"]
>[number];

export type ServiceRow = {
  rowKey: string;
  instance: ServiceInstance;
  state: string;
  /** Helm/pod detail from overview when the row is not running. */
  statusHint?: string;
  connection?: InstanceConnection;
  /** ISO time of this instance's last successful backup, if any. */
  lastBackupAt?: string;
  /** Latest CPU / memory / disk sample for this instance, if collected. */
  vitals?: VitalsRow;
};

export function serviceRowKey(inst: { type: string; id: string }) {
  return `${inst.type}-${inst.id}`;
}

export function buildServiceRows(
  instances: ServiceInstance[],
  serviceStates: Record<string, string> | undefined,
  connections: ConnectionsResponse | null,
  statusLoaded: boolean,
  lastBackups?: Record<string, string>,
  vitals?: Record<string, VitalsRow>,
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
      lastBackupAt: lastBackups?.[`${inst.type}/${inst.id}`],
      vitals: vitals?.[`${inst.type}/${inst.id}`],
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
  if (conn.mongodb) out.push({ label: "MongoDB", value: conn.mongodb });
  if (conn.redis) out.push({ label: "Redis", value: conn.redis });
  if (conn.s3_http) out.push({ label: "S3 HTTP", value: conn.s3_http });
  if (conn.qdrant_http)
    out.push({ label: "Qdrant REST", value: conn.qdrant_http });
  if (conn.qdrant_grpc)
    out.push({ label: "Qdrant gRPC", value: conn.qdrant_grpc });
  if (conn.searxng_http)
    out.push({ label: "SearXNG", value: conn.searxng_http });
  return out;
}

/** Short label for tables; full credential is `connectionUse`. */
export function connectionPreview(conn?: InstanceConnection): string | null {
  const use = connectionUse(conn);
  if (!use) return null;
  const first = use.split("\n")[0] ?? use;
  if (use.includes("\n")) {
    return `${first} …`;
  }
  return first;
}

/** Admin-shareable credential block (one per service). */
export function connectionUse(conn?: InstanceConnection): string | null {
  if (conn?.use?.trim()) return conn.use.trim();
  const entries = connectionEntries(conn);
  return entries[0]?.value ?? null;
}

/** Copy-ready block for details / share dialogs. */
export function connectionDetailsBlob(conn?: InstanceConnection): string | null {
  return connectionUse(conn);
}

export function stateBadgeVariant(state: string) {
  if (state === "running") return "success" as const;
  if (state === "starting" || state === "stopping" || state === "pending") {
    return "warning" as const;
  }
  if (state === "degraded") return "live" as const;
  return "default" as const;
}

/** True while desired state and cluster state may still be converging. */
export function isTransitionalState(state: string) {
  return (
    state === "starting" ||
    state === "stopping" ||
    state === "pending" ||
    state === "loading"
  );
}

/** Per-row Start/Stop affordance from observed cluster state. */
export function startStopTarget(state: string): "start" | "stop" | null {
  if (state === "stopped") return "start";
  if (state === "running" || state === "degraded") return "stop";
  return null;
}
