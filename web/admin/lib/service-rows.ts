import type { ConnectionsResponse, ServiceInstance } from "@/lib/api";
import type { ConnectionAudience, VitalsRow } from "@/lib/types";

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

/** Flat connection fields for one reachability audience. */
export type ResolvedConnection = InstanceConnection;

export function resolveConnection(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): ResolvedConnection | undefined {
  if (!conn) return undefined;
  if (audience !== "in_cluster" || !conn.in_cluster) {
    return conn;
  }
  const inner = conn.in_cluster;
  return {
    ...conn,
    use: inner.use ?? conn.use,
    postgres: inner.postgres ?? conn.postgres,
    mysql: inner.mysql ?? conn.mysql,
    mongodb: inner.mongodb ?? conn.mongodb,
    redis: inner.redis ?? conn.redis,
    s3_http: inner.s3_http ?? conn.s3_http,
    qdrant_http: inner.qdrant_http ?? conn.qdrant_http,
    qdrant_grpc: inner.qdrant_grpc ?? conn.qdrant_grpc,
    searxng_http: inner.searxng_http ?? conn.searxng_http,
    kafka_bootstrap: inner.kafka_bootstrap ?? conn.kafka_bootstrap,
    schema_registry_http:
      inner.schema_registry_http ?? conn.schema_registry_http,
    unimeter_api: inner.unimeter_api ?? conn.unimeter_api,
    unimeter_health_http:
      inner.unimeter_health_http ?? conn.unimeter_health_http,
    bunqueue_tcp: inner.bunqueue_tcp ?? conn.bunqueue_tcp,
    bunqueue_health_http:
      inner.bunqueue_health_http ?? conn.bunqueue_health_http,
    public_path: inner.host ?? conn.public_path,
  };
}

export function connectionEntries(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): ConnectionEntry[] {
  const view = resolveConnection(conn, audience);
  if (!view) return [];
  const out: ConnectionEntry[] = [];
  if (view.postgres) out.push({ label: "PostgreSQL", value: view.postgres });
  if (view.mysql) out.push({ label: "MySQL", value: view.mysql });
  if (view.mongodb) out.push({ label: "MongoDB", value: view.mongodb });
  if (view.redis) out.push({ label: "Redis", value: view.redis });
  if (view.s3_http) out.push({ label: "S3 HTTP", value: view.s3_http });
  if (view.qdrant_http)
    out.push({ label: "Qdrant REST", value: view.qdrant_http });
  if (view.qdrant_grpc)
    out.push({ label: "Qdrant gRPC", value: view.qdrant_grpc });
  if (view.searxng_http)
    out.push({ label: "SearXNG", value: view.searxng_http });
  if (view.kafka_bootstrap)
    out.push({ label: "Kafka bootstrap", value: view.kafka_bootstrap });
  if (view.schema_registry_http)
    out.push({
      label: "Schema Registry",
      value: view.schema_registry_http,
    });
  if (view.unimeter_api)
    out.push({ label: "Unimeter API", value: view.unimeter_api });
  if (view.unimeter_health_http)
    out.push({ label: "Unimeter health", value: view.unimeter_health_http });
  if (view.bunqueue_tcp)
    out.push({ label: "bunqueue TCP", value: view.bunqueue_tcp });
  if (view.bunqueue_health_http)
    out.push({ label: "bunqueue health", value: view.bunqueue_health_http });
  if (audience === "in_cluster" && conn?.in_cluster?.host) {
    out.unshift({ label: "Service DNS", value: conn.in_cluster.host });
  }
  return out;
}

/** Short label for tables; full credential is `connectionUse`. */
export function connectionPreview(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): string | null {
  const use = connectionUse(conn, audience);
  if (!use) return null;
  const first = use.split("\n")[0] ?? use;
  if (use.includes("\n")) {
    return `${first} …`;
  }
  return first;
}

/** Admin-shareable credential block (one per service). */
export function connectionUse(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): string | null {
  const view = resolveConnection(conn, audience);
  if (view?.use?.trim()) return view.use.trim();
  const entries = connectionEntries(conn, audience);
  return entries[0]?.value ?? null;
}

/** Copy-ready block for details / share dialogs. */
export function connectionDetailsBlob(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): string | null {
  const view = resolveConnection(conn, audience);
  const use = view?.use?.trim();
  if (use) return use;
  const entries = connectionEntries(conn, audience);
  if (entries.length === 0) return null;
  return entries.map((e) => `${e.label}: ${e.value}`).join("\n");
}

/** KEY=value lines in `use` blocks; not query params inside connection URLs. */
function useLineEnvKey(line: string): string | null {
  const eq = line.indexOf("=");
  if (eq <= 0) return null;
  const key = line.slice(0, eq).trim();
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) return null;
  return key;
}

/** Structured rows for the admin UI (every remote connection field). */
export function connectionCredentialLines(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): ConnectionEntry[] {
  const view = resolveConnection(conn, audience);
  const use = view?.use?.trim();
  if (use) {
    const out: ConnectionEntry[] = [];
    const typed = connectionEntries(conn, audience);
    for (const line of use.split("\n")) {
      const t = line.trim();
      if (!t) continue;
      const envKey = useLineEnvKey(t);
      if (envKey) {
        out.push({
          label: envKey,
          value: t.slice(envKey.length + 1).trim(),
        });
      } else {
        const match = typed.find((e) => e.value === t);
        out.push(match ?? { label: "Connection", value: t });
      }
    }
    if (out.length > 0) return out;
  }
  return connectionEntries(conn, audience);
}

/** True when URLs are present but expected secret env lines are absent. */
export function connectionCredentialsIncomplete(
  conn?: InstanceConnection,
  audience: ConnectionAudience = "edge",
): boolean {
  const view = resolveConnection(conn, audience);
  const use = view?.use?.trim() ?? "";
  if (!use) return false;
  const has = (key: string) => use.split("\n").some((l) => l.startsWith(`${key}=`));
  if (view?.s3_http && (!has("AWS_ACCESS_KEY_ID") || !has("AWS_SECRET_ACCESS_KEY"))) {
    return true;
  }
  if (view?.qdrant_http && !has("QDRANT_API_KEY") && !has("api-key")) {
    return true;
  }
  return false;
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
