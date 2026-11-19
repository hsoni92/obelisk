"use client";

import { PageHeader } from "@/components/layout/page-header";
import { NewInstanceButton } from "@/components/services/create-instance-dialog";
import { ServicesFilters } from "@/components/services/services-filters";
import { DiagnosticsResultsDialog } from "@/components/services/diagnostics-results-dialog";
import {
  runServicesBulkAction,
  ServicesBulkActionsMenu,
  useBulkActionCounts,
} from "@/components/services/services-bulk-actions";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
  useRunDiagnosticsMutation,
  useDeleteInstanceMutation,
  usePatchInstanceMutation,
  type ServiceInstance,
} from "@/lib/api";
import type { DiagnosticInstanceResult } from "@/lib/types";
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
  const [terminateConfirm, setTerminateConfirm] = useState(false);
  const [terminateBusy, setTerminateBusy] = useState(false);
  const [configureTarget, setConfigureTarget] =
    useState<ServiceInstance | null>(null);
  const [diagOpen, setDiagOpen] = useState(false);
  const [diagRunning, setDiagRunning] = useState(false);
  const [diagResults, setDiagResults] =
    useState<DiagnosticInstanceResult[] | null>(null);
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
  const [runDiagnostics] = useRunDiagnosticsMutation();

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
  const diagnoseCount = useMemo(() => {
    let n = 0;
    for (const row of tableRows) {
      if (!selected.has(row.rowKey)) continue;
      if (row.instance.enabled) n += 1;
    }
    return n;
  }, [tableRows, selected]);

  const selectedRows = useMemo(
    () => tableRows.filter((r) => selected.has(r.rowKey)),
    [tableRows, selected],
  );

  async function runBulk(action: "start" | "stop") {
    setBulkBusy(true);
    try {
      await runServicesBulkAction(
        tableRows,
        selected,
        action,
        setEnabled,
        () => setSelected(new Set()),
      );
    } finally {
      setBulkBusy(false);
    }
  }

  async function runBulkTerminate() {
    if (selectedRows.length === 0) {
      toast.warning("Nothing to terminate", "Select one or more services first.");
      return;
    }
    setTerminateConfirm(true);
  }

  async function confirmBulkTerminate() {
    setTerminateBusy(true);
    setBulkBusy(true);
    let ok = 0;
    let fail = 0;
    for (const row of selectedRows) {
      const key = row.rowKey;
      setBusy(key);
      try {
        await deleteInstance({
          id: row.instance.id,
          type: row.instance.type,
        }).unwrap();
        ok += 1;
        setSelected((prev) => {
          const next = new Set(prev);
          next.delete(key);
          return next;
        });
      } catch (e) {
        fail += 1;
        toast.error(
          `Terminate failed (${row.instance.type}/${row.instance.id})`,
          rtkErrorMessage(e),
        );
      } finally {
        setBusy(null);
      }
    }
    setTerminateBusy(false);
    setBulkBusy(false);
    setTerminateConfirm(false);
    if (ok > 0 && fail === 0) {
      toast.success(
        "Instances terminated",
        `${ok} removed from the cluster (Helm releases and PVCs).`,
      );
      setSelected(new Set());
    } else if (ok > 0 && fail > 0) {
      toast.warning("Partially terminated", `${ok} removed, ${fail} failed.`);
    }
  }

  async function runConnectionDiagnostics() {
    const instances = tableRows
      .filter((r) => selected.has(r.rowKey) && r.instance.enabled)
      .map((r) => ({ type: r.instance.type, id: r.instance.id }));
    if (instances.length === 0) {
      toast.warning(
        "Nothing to test",
        "Select enabled services that expose connection URLs.",
      );
      return;
    }
    setDiagOpen(true);
    setDiagResults(null);
    setDiagRunning(true);
    setBulkBusy(true);
    try {
      const res = await runDiagnostics({ instances }).unwrap();
      setDiagResults(res.results);
      const failed = res.results.filter((r) => !r.ok && !r.skipped).length;
      const passed = res.results.filter((r) => r.ok && !r.skipped).length;
      if (failed === 0 && passed > 0) {
        toast.success(
          "Connections OK",
          `${passed} endpoint${passed === 1 ? "" : "s"} reachable.`,
        );
      } else if (failed > 0) {
        toast.warning(
          "Some checks failed",
          `${failed} of ${res.results.length} could not use the published URL.`,
        );
      }
    } catch (e) {
      toast.error("Diagnostics failed", rtkErrorMessage(e));
      setDiagOpen(false);
    } finally {
      setDiagRunning(false);
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
      <DiagnosticsResultsDialog
        open={diagOpen}
        onOpenChange={setDiagOpen}
        results={diagResults}
        running={diagRunning}
      />

      <AlertDialog
        open={terminateConfirm}
        onOpenChange={(o) => !terminateBusy && setTerminateConfirm(o)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Terminate {selectedRows.length} instance
              {selectedRows.length === 1 ? "" : "s"}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This removes each instance from the appliance config, uninstalls
              its Helm release, and deletes local volume data (PVCs). It cannot
              be undone. Use <strong>Stop</strong> if you only want to scale
              down and keep data.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={terminateBusy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={terminateBusy || selectedRows.length === 0}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault();
                void confirmBulkTerminate();
              }}
            >
              {terminateBusy ? "Terminating…" : "Terminate permanently"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

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
              diagnoseCount={diagnoseCount}
              busy={bulkBusy}
              disabled={busy !== null}
              onStart={() => void runBulk("start")}
              onStop={() => void runBulk("stop")}
              onDiagnose={() => void runConnectionDiagnostics()}
              onTerminate={() => void runBulkTerminate()}
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
          onClearFilters={() => {
            resetServiceFilters();
            router.replace("/services", { scroll: false });
          }}
          onConfigure={setConfigureTarget}
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
