"use client";

import { PageHeader } from "@/components/layout/page-header";
import { NewInstanceButton } from "@/components/services/create-instance-dialog";
import { ServicesFilters } from "@/components/services/services-filters";
import { ServicesTable, ServicesTableSkeleton } from "@/components/services/services-table";
import {
  ConfigureInstanceDialog,
  type ConfigureInstanceSave,
} from "@/components/services/configure-instance-dialog";
import {
  rtkErrorMessage,
  statusPollOptions,
  useCreateInstanceMutation,
  useGetBackupConfigQuery,
  useGetConnectionsQuery,
  useGetStatusQuery,
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
  const [configureTarget, setConfigureTarget] =
    useState<ServiceInstance | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
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
  } = useGetStatusQuery(undefined, statusPollOptions);
  const { data: connections, error: connectionsError } = useGetConnectionsQuery(
    undefined,
    statusPollOptions,
  );

  const { data: backupConfig } = useGetBackupConfigQuery(undefined, {
    pollingInterval: 30_000,
  });

  const [createInstance] = useCreateInstanceMutation();
  const [patchInstance] = usePatchInstanceMutation();

  const queryError = statusError ?? connectionsError;
  const error =
    actionError ?? (queryError ? rtkErrorMessage(queryError) : null);

  const allTableRows = useMemo(
    () =>
      buildServiceRows(
        status?.instances ?? [],
        status?.serviceStates,
        connections ?? null,
        !statusLoading && status !== undefined,
        backupConfig?.last_backup_by_instance,
      ),
    [status, connections, statusLoading, backupConfig],
  );

  const tableRows = useMemo(
    () => applyFilters(allTableRows, serviceFilters, servicesFilterConfig),
    [allTableRows, serviceFilters],
  );

  async function addInstance(type: (typeof TYPES)[number], name: string) {
    setActionError(null);
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
    setActionError(null);
    try {
      await patchInstance({
        id: inst.id,
        body: { enabled, type: inst.type },
      }).unwrap();
      return true;
    } catch (e) {
      setActionError(rtkErrorMessage(e));
      return false;
    } finally {
      setBusy(null);
    }
  }

  async function saveConfiguration(id: string, patch: ConfigureInstanceSave) {
    const inst = allTableRows.find((r) => r.instance.id === id)?.instance;
    setBusy(inst ? serviceRowKey(inst) : id);
    setActionError(null);
    try {
      await patchInstance({
        id,
        body: { ...patch, type: inst?.type },
      }).unwrap();
    } catch (e) {
      setActionError(rtkErrorMessage(e));
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

      {error && (
        <p
          className="text-xs font-medium text-foreground bg-broadcast-tint rounded-full px-3 py-1 mb-3 inline-block"
          role="alert"
        >
          {error}
        </p>
      )}

      <div className="space-y-4">
        <ServicesFilters />
        <ServicesTable
          loading={statusLoading}
          rows={tableRows}
          totalCount={allTableRows.length}
          filtered={serviceFilterCount > 0}
          busyKey={busy}
          onClearFilters={() => {
            resetServiceFilters();
            router.replace("/services", { scroll: false });
          }}
          onToggle={setEnabled}
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