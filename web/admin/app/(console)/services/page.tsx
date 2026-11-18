"use client";

import { PageHeader } from "@/components/layout/page-header";
import { NewInstanceButton } from "@/components/services/create-instance-dialog";
import { ServicesFilters } from "@/components/services/services-filters";
import {
  runServicesBulkAction,
  ServicesBulkActionsMenu,
  useBulkActionCounts,
  type RowPending,
} from "@/components/services/services-bulk-actions";
import {
  ServicesTable,
  ServicesTableSkeleton,
} from "@/components/services/services-table";
import {
  ConfigureInstanceDialog,
  type ConfigureInstanceSave,
} from "@/components/services/configure-instance-dialog";
import {
  rtkErrorMessage,
  rtkQueryLoadFailed,
  statusPollOptions,
  useCreateInstanceMutation,
  useGetBackupConfigQuery,
  useGetConnectionsQuery,
  useGetOverviewQuery,
  overviewPollOptions,
  useGetStatusQuery,
  useDeleteInstanceMutation,
  usePatchInstanceMutation,
  type ServiceInstance,
} from "@/lib/api";
import {
  applyFilters,
  SERVICES_SURFACE,
  servicesFilterConfig,
} from "@/lib/filters";
import { buildServiceRows, serviceRowKey } from "@/lib/service-rows";
import { useSurfaceFilters } from "@/lib/store/hooks";
import { toast } from "@/lib/toast";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useMemo, useState } from "react";

const TYPES = [
  "postgres",
  "mysql",
  "mongodb",
  "redis",
  "rustfs",
  "qdrant",
  "searxng",
] as const;

function ServicesPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [busy, setBusy] = useState<string | null>(null);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [bulkPending, setBulkPending] = useState<Record<string, RowPending>>({});
  const [configureTarget, setConfigureTarget] =
    useState<ServiceInstance | null>(null);
  const {
    filters: serviceFilters,
    activeCount: serviceFilterCount,
    setQuery: setServiceQuery,
    reset: resetServiceFilters,
  } = useSurfaceFilters(SERVICES_SURFACE);

  const urlQ = searchParams.get("q") ?? "";
  useEffect(() => {
    if (urlQ !== serviceFilters.query) {
      setServiceQuery(urlQ);
    }
  }, [urlQ, serviceFilters.query, setServiceQuery]);

  const {
    data: status,
    error: statusError,
    isLoading: statusLoading,
    isFetching: statusFetching,
  } = useGetStatusQuery(undefined, statusPollOptions);
  const {
    data: connections,
    error: connectionsError,
    isFetching: connectionsFetching,
  } = useGetConnectionsQuery(undefined, statusPollOptions);

  const { data: backupConfig } = useGetBackupConfigQuery(undefined, {
    pollingInterval: 30_000,
  });

  // Same query args as the Overview page, so the cache entry is shared.
  const { data: overview } = useGetOverviewQuery(
    { hours: 6 },
    overviewPollOptions,
  );
  const vitalsByKey = useMemo(
    () =>
      Object.fromEntries((overview?.vitals?.rows ?? []).map((r) => [r.key, r])),
    [overview?.vitals?.rows],
  );

  const statusHints = useMemo(() => {
    const hints: Record<string, string> = {};
    for (const c of overview?.components ?? []) {
      if (c.kind !== "instance" || !c.type || !c.instance_id) continue;
      const parts = [c.reason, c.helm_status ? `helm: ${c.helm_status}` : ""]
        .filter(Boolean)
        .join(" · ");
      if (parts) {
        hints[`${c.type}-${c.instance_id}`] = parts;
      }
    }
    return hints;
  }, [overview?.components]);

  const [createInstance] = useCreateInstanceMutation();
  const [patchInstance] = usePatchInstanceMutation();
  const [deleteInstance] = useDeleteInstanceMutation();

  useEffect(() => {
    if (
      rtkQueryLoadFailed(
        statusError,
        status,
        statusLoading,
        statusFetching,
      )
    ) {
      toast.error("Could not load services", rtkErrorMessage(statusError));
      return;
    }
    if (
      rtkQueryLoadFailed(
        connectionsError,
        connections,
        statusLoading,
        connectionsFetching,
      )
    ) {
      toast.error("Could not load services", rtkErrorMessage(connectionsError));
    }
  }, [
    statusError,
    connectionsError,
    status,
    connections,
    statusLoading,
    statusFetching,
    connectionsFetching,
  ]);

  const allTableRows = useMemo(
    () =>
      buildServiceRows(
        status?.instances ?? [],
        status?.serviceStates,
        connections ?? null,
        !statusLoading && status !== undefined,
        backupConfig?.last_backup_by_instance,
        vitalsByKey,
      ),
    [status, connections, statusLoading, backupConfig, vitalsByKey],
  );

  const tableRows = useMemo(() => {
    const withHints = allTableRows.map((row) => ({
      ...row,
      statusHint: statusHints[row.rowKey],
    }));
    return applyFilters(withHints, serviceFilters, servicesFilterConfig);
  }, [allTableRows, serviceFilters, statusHints]);

  const bulkCounts = useBulkActionCounts(tableRows, selected);

  useEffect(() => {
    setBulkPending((prev) => {
      let changed = false;
      const next: Record<string, RowPending> = {};
      for (const [key, wait] of Object.entries(prev)) {
        const row = tableRows.find((r) => r.rowKey === key);
        if (row && row.state === wait.from) {
          next[key] = wait;
        } else {
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [tableRows]);

  async function runBulk(action: "start" | "stop") {
    setBulkBusy(true);
    try {
      await runServicesBulkAction(
        tableRows,
        selected,
        action,
        setEnabled,
        (key, pending) => setBulkPending((p) => ({ ...p, [key]: pending })),
        () => setSelected(new Set()),
      );
    } finally {
      setBulkBusy(false);
    }
  }

  async function addInstance(type: (typeof TYPES)[number], name: string) {
    const res = await createInstance({
      type,
      display_name: name,
    }).unwrap();
    return res;
  }

  /** Resolves true when the change was accepted, false if the request failed. */
  async function setEnabled(
    inst: ServiceInstance,
    enabled: boolean,
  ): Promise<boolean> {
    const key = serviceRowKey(inst);
    setBusy(key);
    try {
      await patchInstance({
        id: inst.id,
        body: { enabled, type: inst.type },
      }).unwrap();
      return true;
    } catch (e) {
      toast.error(rtkErrorMessage(e));
      return false;
    } finally {
      setBusy(null);
    }
  }

  async function terminateInstance(inst: ServiceInstance): Promise<boolean> {
    const key = serviceRowKey(inst);
    setBusy(key);
    try {
      await deleteInstance({ id: inst.id, type: inst.type }).unwrap();
      toast.success(
        "Instance terminated",
        `${inst.type}/${inst.id} removed from the cluster (Helm release and PVCs).`,
      );
      setSelected((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
      return true;
    } catch (e) {
      toast.error("Terminate failed", rtkErrorMessage(e));
      return false;
    } finally {
      setBusy(null);
    }
  }

  async function saveConfiguration(id: string, patch: ConfigureInstanceSave) {
    const inst = allTableRows.find((r) => r.instance.id === id)?.instance;
    setBusy(inst ? serviceRowKey(inst) : id);
    try {
      await patchInstance({
        id,
        body: { ...patch, type: inst?.type },
      }).unwrap();
      toast.success("Instance updated");
    } catch (e) {
      toast.error(rtkErrorMessage(e));
    } finally {
      setBusy(null);
    }
  }

  return (
    <>
      {configureTarget && (
        <ConfigureInstanceDialog
          instance={configureTarget}
          open={!!configureTarget}
          onOpenChange={(open) => !open && setConfigureTarget(null)}
          onSave={(patch) => saveConfiguration(configureTarget.id, patch)}
        />
      )}

      <PageHeader
        section="services"
        action={<NewInstanceButton onCreate={addInstance} />}
      />

      <div className="space-y-4">
        <ServicesFilters
          trailing={
            <ServicesBulkActionsMenu
              selectedCount={selected.size}
              startCount={bulkCounts.start}
              stopCount={bulkCounts.stop}
              busy={bulkBusy}
              disabled={busy !== null}
              onStart={() => void runBulk("start")}
              onStop={() => void runBulk("stop")}
              onClear={() => setSelected(new Set())}
            />
          }
        />
        <ServicesTable
          loading={statusLoading}
          rows={tableRows}
          totalCount={allTableRows.length}
          filtered={serviceFilterCount > 0}
          busyKey={busy}
          selected={selected}
          onSelectedChange={setSelected}
          selectionDisabled={bulkBusy}
          bulkPending={bulkPending}
          onClearFilters={() => {
            resetServiceFilters();
            router.replace("/services", { scroll: false });
          }}
          onToggle={setEnabled}
          onConfigure={setConfigureTarget}
          onTerminate={(inst) => terminateInstance(inst)}
        />
      </div>
    </>
  );
}

export default function ServicesPage() {
  return (
    <Suspense
      fallback={
        <>
          <PageHeader section="services" />
          <ServicesTableSkeleton />
        </>
      }
    >
      <ServicesPageContent />
    </Suspense>
  );
}
