import type {
  BackupConfigResponse,
  BackupTargetResponse,
  BlueprintAuthorizeResponse,
  BlueprintStatus,
  GitHubHealth,
  SystemHealthReport,
  ConnectionsResponse,
  DiagnosticsRunResponse,
  CreateInstanceResponse,
  MailConfigResponse,
  MCPConfigResponse,
  OpenRouterConfigResponse,
  RestoreChainState,
  ServiceInstance,
  ActivityHistoryResponse,
  OverviewResponse,
  StatusResponse,
  InstanceDatabasesResponse,
  InstanceBackupLineageResponse,
  BackupCatalogResponse,
  CloneBackupResponse,
  ProjectView,
  TailscaleStatusResponse,
  FederationResponse,
  ConnectivityResponse,
  UpdateStatus,
  UpdateApply,
} from "@/lib/types";
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { obeliskApiBaseUrl } from "./api-base";

const POLL_MS = 3000;

export const obeliskApi = createApi({
  reducerPath: "obeliskApi",
  baseQuery: fetchBaseQuery({
    baseUrl: obeliskApiBaseUrl,
    credentials: "include",
    prepareHeaders: (headers) => {
      headers.set("Accept", "application/json");
      return headers;
    },
  }),
  tagTypes: ["Status", "Connections", "BackupTarget", "BackupConfig", "BackupLineage", "BackupCatalog", "Overview", "OpenRouter", "Mail", "MCP", "Blueprint", "Restore", "Projects", "Tailscale", "Deployment", "Federation", "Instances"],
  endpoints: (builder) => ({
    getStatus: builder.query<StatusResponse, void>({
      query: () => "/status",
      providesTags: ["Status"],
    }),
    getOverview: builder.query<OverviewResponse, { hours?: number } | void>({
      query: (arg) => {
        const hours = arg && "hours" in arg && arg.hours ? arg.hours : 6;
        return `/overview?hours=${hours}`;
      },
      providesTags: ["Overview"],
    }),
    getActivityHistory: builder.query<
      ActivityHistoryResponse,
      { hours?: number } | void
    >({
      query: (arg) => {
        const hours = arg && "hours" in arg && arg.hours ? arg.hours : 6;
        return `/activity/history?hours=${hours}`;
      },
      providesTags: ["Overview"],
      keepUnusedDataFor: 60,
    }),
    getConnections: builder.query<ConnectionsResponse, void>({
      query: () => "/connections",
      providesTags: ["Connections"],
    }),
    runDiagnostics: builder.mutation<
      DiagnosticsRunResponse,
      { instances: { type: string; id: string }[] }
    >({
      query: (body) => ({
        url: "/diagnostics/run",
        method: "POST",
        body,
      }),
    }),
    getBackupTarget: builder.query<BackupTargetResponse, void>({
      query: () => "/backup/target",
      providesTags: ["BackupTarget"],
    }),
    getBackupConfig: builder.query<BackupConfigResponse, void>({
      query: () => "/backup/config",
      providesTags: ["BackupConfig"],
      keepUnusedDataFor: 30,
    }),
    patchBackupPolicy: builder.mutation<
      BackupConfigResponse,
      {
        schedule?: string;
        replication_factor?: number;
        data_retention_days?: number;
        config_keep_count?: number;
        include_config?: boolean;
      }
    >({
      query: (body) => ({
        url: "/backup/policy",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    postBackupDestination: builder.mutation<
      BackupConfigResponse,
      {
        label: string;
        provider?: string;
        endpoint: string;
        enabled?: boolean;
        credentials?: { access_key_id: string; secret_access_key: string };
      }
    >({
      query: (body) => ({
        url: "/backup/destinations",
        method: "POST",
        body,
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    patchBackupDestination: builder.mutation<
      BackupConfigResponse,
      {
        id: string;
        body: {
          label?: string;
          provider?: string;
          endpoint?: string;
          enabled?: boolean;
          credentials?: { access_key_id: string; secret_access_key: string };
        };
      }
    >({
      query: ({ id, body }) => ({
        url: `/backup/destinations/${encodeURIComponent(id)}`,
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    deleteBackupDestination: builder.mutation<
      BackupConfigResponse,
      {
        id: string;
        acknowledge_primary_removal?: boolean;
        promote_destination_id?: string;
      }
    >({
      query: ({ id, acknowledge_primary_removal, promote_destination_id }) => ({
        url: `/backup/destinations/${encodeURIComponent(id)}`,
        method: "DELETE",
        body: {
          acknowledge_primary_removal: acknowledge_primary_removal ?? false,
          promote_destination_id: promote_destination_id,
        },
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    promoteBackupDestination: builder.mutation<BackupConfigResponse, { id: string }>({
      query: ({ id }) => ({
        url: `/backup/destinations/${encodeURIComponent(id)}/promote-primary`,
        method: "POST",
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    testBackupDestination: builder.mutation<BackupConfigResponse, { id: string }>({
      query: ({ id }) => ({
        url: `/backup/destinations/${encodeURIComponent(id)}/test`,
        method: "POST",
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    testAllBackupDestinations: builder.mutation<BackupConfigResponse, void>({
      query: () => ({
        url: "/backup/destinations/test-all",
        method: "POST",
      }),
      invalidatesTags: ["BackupConfig", "BackupTarget", "Status"],
    }),
    getProjects: builder.query<ProjectView[], void>({
      query: () => "/projects",
      providesTags: ["Projects"],
    }),
    createProject: builder.mutation<
      ProjectView,
      { name: string; instance_ids?: string[]; discovery_enabled?: boolean }
    >({
      query: (body) => ({
        url: "/projects",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Projects"],
    }),
    patchProject: builder.mutation<
      ProjectView,
      {
        id: string;
        body: {
          name?: string;
          instance_ids?: string[];
          discovery_enabled?: boolean;
        };
      }
    >({
      query: ({ id, body }) => ({
        url: `/projects/${encodeURIComponent(id)}`,
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Projects"],
    }),
    deleteProject: builder.mutation<{ deleted: string }, string>({
      query: (id) => ({
        url: `/projects/${encodeURIComponent(id)}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Projects"],
    }),
    rotateProjectDiscoveryToken: builder.mutation<
      { token: string },
      string
    >({
      query: (id) => ({
        url: `/projects/${encodeURIComponent(id)}/rotate-discovery-token`,
        method: "POST",
      }),
      invalidatesTags: ["Projects"],
    }),
    createInstance: builder.mutation<
      CreateInstanceResponse,
      {
        type: string;
        display_name: string;
        from_backup?: {
          source_type: string;
          source_instance_id: string;
          backup_id?: string;
        };
      }
    >({
      query: (body) => ({
        url: "/instances",
        method: "POST",
        body,
      }),
      invalidatesTags: [
        "Status",
        "Connections",
        "Overview",
        "BackupCatalog",
        "BackupLineage",
      ],
    }),
    patchInstance: builder.mutation<
      unknown,
      {
        id: string;
        body: {
          type?: string;
          display_name?: string;
          enabled?: boolean;
          backup?: {
            enabled?: boolean;
            policy_override?: {
              schedule?: string;
              replication_factor?: number;
              data_retention_days?: number;
            };
            clear_policy_override?: boolean;
          };
          credentials?: ServiceInstance["credentials"];
        };
      }
    >({
      query: ({ id, body }) => ({
        url: `/instances/${id}`,
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Status", "Connections", "Overview"],
    }),
    deleteInstance: builder.mutation<
      unknown,
      { id: string; type: string; delete_backups?: boolean }
    >({
      query: ({ id, type, delete_backups }) => {
        const params = new URLSearchParams({ type });
        if (delete_backups) {
          params.set("delete_backups", "true");
        }
        return {
          url: `/instances/${encodeURIComponent(id)}?${params.toString()}`,
          method: "DELETE",
        };
      },
      invalidatesTags: ["Status", "Connections", "Overview", "BackupLineage", "BackupCatalog", "BackupConfig"],
    }),
    getInstanceBackupLineage: builder.query<
      InstanceBackupLineageResponse,
      { id: string; type: string }
    >({
      query: ({ id, type }) =>
        `/instances/${encodeURIComponent(id)}/backups?type=${encodeURIComponent(type)}`,
      providesTags: (_r, _e, arg) => [
        { type: "BackupLineage", id: `${arg.type}/${arg.id}` },
      ],
    }),
    restartInstance: builder.mutation<
      {
        status: string;
        method: string;
        message: string;
        state_key?: string;
      },
      { id: string; type: string }
    >({
      query: ({ id, type }) => ({
        url: `/instances/${encodeURIComponent(id)}/restart?type=${encodeURIComponent(type)}`,
        method: "POST",
      }),
      invalidatesTags: ["Status", "Connections", "Overview"],
    }),
    runInstanceBackup: builder.mutation<
      { status: string; backups?: string[]; message?: string },
      { id: string; type: string }
    >({
      query: ({ id, type }) => ({
        url: `/instances/${encodeURIComponent(id)}/backup/run?type=${encodeURIComponent(type)}`,
        method: "POST",
      }),
      invalidatesTags: (_r, _e, arg) => [
        "BackupConfig",
        "BackupCatalog",
        "Status",
        { type: "BackupLineage", id: `${arg.type}/${arg.id}` },
      ],
    }),
    restoreInstanceActiveBackup: builder.mutation<
      { status: string; backup_id?: string },
      { id: string; type: string }
    >({
      query: ({ id, type }) => ({
        url: `/instances/${encodeURIComponent(id)}/backup/restore-active?type=${encodeURIComponent(type)}`,
        method: "POST",
      }),
      invalidatesTags: ["Status", { type: "BackupLineage" }],
    }),
    deleteInstanceBackup: builder.mutation<
      { status: string; deleted?: string[] },
      { id: string; type: string; backupId: string; confirm: string }
    >({
      query: ({ id, type, backupId, confirm }) => ({
        url: `/instances/${encodeURIComponent(id)}/backups/${encodeURIComponent(backupId)}?type=${encodeURIComponent(type)}`,
        method: "DELETE",
        body: { confirm },
      }),
      invalidatesTags: (_r, _e, arg) => [
        "BackupConfig",
        "BackupCatalog",
        { type: "BackupLineage", id: `${arg.type}/${arg.id}` },
      ],
    }),
    deleteInstanceBackupForest: builder.mutation<
      { status: string; deleted?: string[]; message?: string },
      { id: string; type: string; confirm: string }
    >({
      query: ({ id, type, confirm }) => ({
        url: `/instances/${encodeURIComponent(id)}/backups?type=${encodeURIComponent(type)}`,
        method: "DELETE",
        body: { confirm },
      }),
      invalidatesTags: (_r, _e, arg) => [
        "BackupConfig",
        "BackupCatalog",
        { type: "BackupLineage", id: `${arg.type}/${arg.id}` },
      ],
    }),
    cloneBackup: builder.mutation<
      CloneBackupResponse,
      {
        source_type: string;
        source_instance_id: string;
        scope?: "snapshot" | "lineage";
        backup_id?: string;
        label?: string;
      }
    >({
      query: (body) => ({
        url: "/backups/clone",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Status", "Connections", "BackupCatalog", "BackupLineage", "Overview"],
    }),
    getBackupCatalog: builder.query<BackupCatalogResponse, void>({
      query: () => "/backup/catalog",
      providesTags: ["BackupCatalog"],
    }),
    rebuildBackupIndex: builder.mutation<{ status: string; nodes_added: number }, void>({
      query: () => ({
        url: "/backup/index/rebuild",
        method: "POST",
      }),
      invalidatesTags: ["BackupLineage", "BackupCatalog", "BackupConfig"],
    }),
    getInstanceDatabases: builder.query<
      InstanceDatabasesResponse,
      { id: string; type: string }
    >({
      query: ({ id, type }) =>
        `/instances/${encodeURIComponent(id)}/databases?type=${encodeURIComponent(type)}`,
      providesTags: ["Connections"],
    }),
    addInstanceDatabase: builder.mutation<
      unknown,
      { id: string; type: string; name: string }
    >({
      query: ({ id, type, name }) => ({
        url: `/instances/${encodeURIComponent(id)}/databases?type=${encodeURIComponent(type)}`,
        method: "POST",
        body: { name },
      }),
      invalidatesTags: ["Connections", "Status"],
    }),
    removeInstanceDatabase: builder.mutation<
      unknown,
      { id: string; type: string; name: string }
    >({
      query: ({ id, type, name }) => ({
        url: `/instances/${encodeURIComponent(id)}/databases/${encodeURIComponent(name)}?type=${encodeURIComponent(type)}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Connections", "Status"],
    }),
    patchBackupTarget: builder.mutation<
      BackupTargetResponse,
      {
        remote?: { endpoint: string };
        credentials?: { access_key_id: string; secret_access_key: string };
      }
    >({
      query: (body) => ({
        url: "/backup/target",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["BackupTarget", "Status"],
    }),
    testBackupRemote: builder.mutation<BackupTargetResponse, void>({
      query: () => ({
        url: "/backup/test",
        method: "POST",
      }),
      invalidatesTags: ["BackupTarget", "BackupConfig", "Status"],
    }),
    runBackup: builder.mutation<
      {
        status: string;
        message?: string;
        object_key?: string;
        uploads?: number;
        note?: string;
        data_objects?: string[];
        data_skipped?: string[];
      },
      void
    >({
      query: () => ({
        url: "/backup/run",
        method: "POST",
      }),
      invalidatesTags: ["Status", "BackupConfig", "BackupLineage", "BackupCatalog"],
    }),
    getOpenRouterConfig: builder.query<OpenRouterConfigResponse, void>({
      query: () => "/openrouter/config",
      providesTags: ["OpenRouter"],
    }),
    patchOpenRouterConfig: builder.mutation<
      OpenRouterConfigResponse | { gateway_token: string },
      { enabled?: boolean; regenerate_gateway_token?: boolean }
    >({
      query: (body) => ({
        url: "/openrouter/config",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["OpenRouter"],
    }),
    postOpenRouterAccount: builder.mutation<
      { account: OpenRouterConfigResponse["accounts"][0] },
      { label: string; api_key: string; enabled?: boolean; notes?: string }
    >({
      query: (body) => ({
        url: "/openrouter/accounts",
        method: "POST",
        body,
      }),
      invalidatesTags: ["OpenRouter"],
    }),
    patchOpenRouterAccount: builder.mutation<
      OpenRouterConfigResponse,
      {
        id: string;
        body: {
          label?: string;
          enabled?: boolean;
          notes?: string;
          api_key?: string;
        };
      }
    >({
      query: ({ id, body }) => ({
        url: `/openrouter/accounts/${id}`,
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["OpenRouter"],
    }),
    deleteOpenRouterAccount: builder.mutation<
      { deleted: string },
      { id: string }
    >({
      query: ({ id }) => ({
        url: `/openrouter/accounts/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: ["OpenRouter"],
    }),
    getMailConfig: builder.query<MailConfigResponse, void>({
      query: () => "/mail/config",
      providesTags: ["Mail"],
    }),
    patchMailConfig: builder.mutation<
      MailConfigResponse | { gateway_token: string },
      { enabled?: boolean; regenerate_gateway_token?: boolean }
    >({
      query: (body) => ({
        url: "/mail/config",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Mail"],
    }),
    postMailAccount: builder.mutation<
      { account: MailConfigResponse["accounts"][0] },
      { label: string; address: string; secret: string; enabled?: boolean; notes?: string }
    >({
      query: (body) => ({
        url: "/mail/accounts",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Mail"],
    }),
    patchMailAccount: builder.mutation<
      MailConfigResponse,
      {
        id: string;
        body: {
          label?: string;
          enabled?: boolean;
          notes?: string;
          secret?: string;
          address?: string;
        };
      }
    >({
      query: ({ id, body }) => ({
        url: `/mail/accounts/${id}`,
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Mail"],
    }),
    deleteMailAccount: builder.mutation<{ deleted: string }, { id: string }>({
      query: ({ id }) => ({
        url: `/mail/accounts/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Mail"],
    }),
    testMail: builder.mutation<
      {
        ok: boolean;
        message?: string;
        error?: string;
        from?: string;
        account_id?: string;
        to?: string;
      },
      { to?: string } | void
    >({
      query: (body) => ({
        url: "/mail/test",
        method: "POST",
        body: body ?? {},
      }),
      invalidatesTags: ["Mail"],
    }),
    getMCPConfig: builder.query<MCPConfigResponse, void>({
      query: () => "/mcp/config",
      providesTags: ["MCP"],
    }),
    patchMCPConfig: builder.mutation<
      MCPConfigResponse | { token: string },
      {
        enabled?: boolean;
        regenerate_token?: boolean;
        tools?: Record<string, boolean>;
      }
    >({
      query: (body) => ({ url: "/mcp/config", method: "PATCH", body }),
      invalidatesTags: ["MCP"],
    }),
    testMCP: builder.mutation<
      { ok: boolean; error?: string; tool_count?: number; search?: string },
      void
    >({
      query: () => ({ url: "/mcp/test", method: "POST", body: {} }),
      invalidatesTags: ["MCP"],
    }),
    getTailscaleStatus: builder.query<TailscaleStatusResponse, void>({
      query: () => "/tailscale/status",
      providesTags: ["Tailscale", "Deployment"],
    }),
    connectTailscale: builder.mutation<
      { ok: boolean; status: TailscaleStatusResponse["status"] },
      { auth_key: string; enable_in_config?: boolean; gateway_role?: boolean }
    >({
      query: (body) => ({
        url: "/tailscale/connect",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Tailscale", "Deployment", "Status", "Overview", "Federation"],
    }),
    patchDeployment: builder.mutation<
      { profile: string; role: string; tailscale: boolean },
      { role?: string; tailscale_enabled?: boolean }
    >({
      query: (body) => ({
        url: "/deployment",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Tailscale", "Deployment", "Status", "Overview", "Federation"],
    }),
    getFederation: builder.query<FederationResponse, void>({
      query: () => "/federation",
      providesTags: ["Federation"],
    }),
    getConnectivity: builder.query<ConnectivityResponse, void>({
      query: () => "/connectivity",
      providesTags: ["Federation"],
    }),
    getUpdateStatus: builder.query<UpdateStatus, { refresh?: boolean } | void>({
      query: (arg) => (arg && arg.refresh ? "/update?refresh=1" : "/update"),
    }),
    applyUpdate: builder.mutation<UpdateApply, void>({
      query: () => ({ url: "/update/apply", method: "POST" }),
    }),
    patchFederation: builder.mutation<
      { ok: boolean; advertise_url: string },
      { advertise_url: string }
    >({
      query: (body) => ({
        url: "/federation",
        method: "PATCH",
        body,
      }),
      invalidatesTags: ["Federation", "Instances", "Status", "Connections", "Overview"],
    }),
    federationAdvertiseNow: builder.mutation<
      { ok: boolean; advertise_status: FederationResponse["advertise_status"] },
      void
    >({
      query: () => ({
        url: "/federation/advertise",
        method: "POST",
      }),
      invalidatesTags: ["Federation", "Instances", "Status", "Connections", "Overview"],
    }),
    federationUnregister: builder.mutation<
      { ok: boolean; advertise_url: string; gateway_warning?: string },
      void
    >({
      query: () => ({
        url: "/federation/unregister",
        method: "POST",
      }),
      invalidatesTags: ["Federation", "Instances", "Status", "Connections", "Overview"],
    }),
    deleteFederationPeer: builder.mutation<{ ok: boolean; peer_id: string }, string>({
      query: (peerId) => ({
        url: `/peers/${encodeURIComponent(peerId)}`,
        method: "DELETE",
      }),
      invalidatesTags: ["Federation", "Instances", "Status", "Connections", "Overview"],
    }),
    getBlueprintStatus: builder.query<BlueprintStatus, void>({
      query: () => "/blueprint/status",
      providesTags: ["Blueprint"],
    }),
    getSystemHealth: builder.query<SystemHealthReport, { fresh?: boolean } | void>({
      query: (arg) => (arg && arg.fresh ? "/system/health?fresh=1" : "/system/health"),
    }),
    getBlueprintGitHubConfig: builder.query<{ client_configured: boolean; scope: string }, void>({
      query: () => "/blueprint/github/config",
    }),
    authorizeGitHub: builder.mutation<
      BlueprintAuthorizeResponse,
      { flow: "code" | "device"; redirect_uri?: string }
    >({
      query: (body) => ({
        url: "/blueprint/github/authorize",
        method: "POST",
        body,
      }),
    }),
    completeGitHubCallback: builder.mutation<
      { status: string; login?: string },
      { code?: string; state?: string; device_code?: string }
    >({
      query: (body) => ({
        url: "/blueprint/github/callback",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Blueprint"],
    }),
    logoutGitHub: builder.mutation<{ status: string }, void>({
      query: () => ({
        url: "/blueprint/github/logout",
        method: "POST",
      }),
      invalidatesTags: ["Blueprint"],
    }),
    onboardBlueprint: builder.mutation<
      BlueprintStatus,
      { repository?: string; branch?: string; passphrase: string }
    >({
      query: (body) => ({
        url: "/blueprint/onboard",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Blueprint", "Status"],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled;
          dispatch(
            obeliskApi.util.upsertQueryData("getBlueprintStatus", undefined, data),
          );
        } catch {
          /* keep prior cache on failure */
        }
      },
    }),
    unlockBlueprint: builder.mutation<BlueprintStatus, { passphrase: string }>({
      query: (body) => ({
        url: "/blueprint/unlock",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Blueprint"],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled;
          dispatch(
            obeliskApi.util.upsertQueryData("getBlueprintStatus", undefined, data),
          );
        } catch {
          /* keep prior cache on failure */
        }
      },
    }),
    lockBlueprint: builder.mutation<BlueprintStatus, void>({
      query: () => ({
        url: "/blueprint/lock",
        method: "POST",
      }),
      invalidatesTags: ["Blueprint"],
    }),
    importBlueprint: builder.mutation<
      BlueprintStatus,
      { passphrase?: string; repository?: string; machine_id_choice?: "adopt" | "keep" }
    >({
      query: (body) => ({
        url: "/blueprint/import",
        method: "POST",
        body,
      }),
      invalidatesTags: [
        "Blueprint",
        "Status",
        "BackupConfig",
        "Overview",
        "Connections",
        "OpenRouter",
        "Mail",
        "MCP",
      ],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled;
          dispatch(
            obeliskApi.util.upsertQueryData("getBlueprintStatus", undefined, data),
          );
        } catch {
          /* keep prior cache on failure */
        }
      },
    }),
    pushBlueprint: builder.mutation<BlueprintStatus, { force?: boolean } | void>({
      query: (body) => ({
        url: "/blueprint/push",
        method: "POST",
        body: body ?? undefined,
      }),
      invalidatesTags: ["Blueprint"],
    }),
    startRestoreChain: builder.mutation<
      RestoreChainState,
      { passphrase: string; repository?: string; machine_id_choice?: "adopt" | "keep" }
    >({
      query: (body) => ({
        url: "/restore/chain",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Restore", "Blueprint", "Status"],
    }),
    getRestoreChain: builder.query<RestoreChainState, void>({
      query: () => "/restore/chain",
      providesTags: ["Restore"],
    }),
    testOpenRouter: builder.mutation<
      {
        ok: boolean;
        status_code?: number;
        body_preview?: string;
        message?: string;
        error?: string;
      },
      { gateway_url?: string } | void
    >({
      query: (arg) => ({
        url:
          arg && "gateway_url" in arg && arg.gateway_url
            ? `/openrouter/test?gateway_url=${encodeURIComponent(arg.gateway_url)}`
            : "/openrouter/test",
        method: "POST",
      }),
      invalidatesTags: ["OpenRouter"],
    }),
  }),
});

export const {
  useGetStatusQuery,
  useLazyGetStatusQuery,
  useGetOverviewQuery,
  useGetActivityHistoryQuery,
  useGetConnectionsQuery,
  useRunDiagnosticsMutation,
  useGetBackupTargetQuery,
  useGetBackupConfigQuery,
  usePatchBackupPolicyMutation,
  usePostBackupDestinationMutation,
  usePatchBackupDestinationMutation,
  useDeleteBackupDestinationMutation,
  usePromoteBackupDestinationMutation,
  useTestBackupDestinationMutation,
  useTestAllBackupDestinationsMutation,
  useCreateInstanceMutation,
  useGetInstanceDatabasesQuery,
  useAddInstanceDatabaseMutation,
  useRemoveInstanceDatabaseMutation,
  usePatchInstanceMutation,
  useRestartInstanceMutation,
  useDeleteInstanceMutation,
  useGetInstanceBackupLineageQuery,
  useRunInstanceBackupMutation,
  useRestoreInstanceActiveBackupMutation,
  useDeleteInstanceBackupMutation,
  useDeleteInstanceBackupForestMutation,
  useCloneBackupMutation,
  useRebuildBackupIndexMutation,
  useGetBackupCatalogQuery,
  usePatchBackupTargetMutation,
  useTestBackupRemoteMutation,
  useRunBackupMutation,
  useGetOpenRouterConfigQuery,
  usePatchOpenRouterConfigMutation,
  usePostOpenRouterAccountMutation,
  usePatchOpenRouterAccountMutation,
  useDeleteOpenRouterAccountMutation,
  useTestOpenRouterMutation,
  useGetMailConfigQuery,
  useGetMCPConfigQuery,
  usePatchMCPConfigMutation,
  useTestMCPMutation,
  usePatchMailConfigMutation,
  usePostMailAccountMutation,
  usePatchMailAccountMutation,
  useDeleteMailAccountMutation,
  useTestMailMutation,
  useGetBlueprintStatusQuery,
  useGetSystemHealthQuery,
  useLazyGetSystemHealthQuery,
  useGetBlueprintGitHubConfigQuery,
  useAuthorizeGitHubMutation,
  useCompleteGitHubCallbackMutation,
  useLogoutGitHubMutation,
  useOnboardBlueprintMutation,
  useUnlockBlueprintMutation,
  useLockBlueprintMutation,
  useImportBlueprintMutation,
  usePushBlueprintMutation,
  useStartRestoreChainMutation,
  useGetRestoreChainQuery,
  useGetProjectsQuery,
  useCreateProjectMutation,
  usePatchProjectMutation,
  useDeleteProjectMutation,
  useRotateProjectDiscoveryTokenMutation,
  useGetTailscaleStatusQuery,
  useConnectTailscaleMutation,
  usePatchDeploymentMutation,
  useGetFederationQuery,
  useGetConnectivityQuery,
  useGetUpdateStatusQuery,
  useApplyUpdateMutation,
  usePatchFederationMutation,
  useFederationAdvertiseNowMutation,
  useFederationUnregisterMutation,
  useDeleteFederationPeerMutation,
} = obeliskApi;

export const tailscalePollOptions = { pollingInterval: 8000 } as const;
export const federationPollOptions = { pollingInterval: 8000 } as const;
export const connectivityPollOptions = { pollingInterval: 8000 } as const;

/** Default polling for live cluster status. */
export const statusPollOptions = { pollingInterval: POLL_MS } as const;

/** Overview polls helm/kubectl; keep slightly slower than /status. */
export const overviewPollOptions = { pollingInterval: 8000 } as const;

/** Activity history is file-backed only — safe to poll while the drawer is open. */
export const activityHistoryPollOptions = { pollingInterval: 8000 } as const;

/** Toast-worthy load failure: no cached data yet (not a refetch abort after start/stop). */
export function rtkQueryLoadFailed(
  error: unknown,
  data: unknown,
  isLoading: boolean,
  isFetching = false,
): boolean {
  if (!error || isLoading || isFetching) return false;
  return data == null;
}

export function rtkErrorMessage(error: unknown): string {
  if (!error || typeof error !== "object") return "request failed";
  if ("data" in error) {
    const data = (error as { data?: unknown }).data;
    if (typeof data === "string" && data.trim()) {
      return data.trim();
    }
    if (typeof data === "object" && data) {
      if ("error" in data && (data as { error: unknown }).error === "duplicate_key") {
        const d = data as { message?: string; existing_label?: string; key_tail?: string };
        return (
          d.message ??
          `Duplicate key (already saved as ${d.existing_label ?? "account"}, tail …${d.key_tail ?? ""})`
        );
      }
      if ("message" in data && (data as { message?: unknown }).message) {
        return String((data as { message: unknown }).message);
      }
      if ("error" in data && typeof (data as { error?: unknown }).error === "string") {
        return (data as { error: string }).error;
      }
    }
  }
  if ("error" in error && typeof (error as { error: unknown }).error === "string") {
    const name = (error as { error: string }).error;
    if (name === "FETCH_ERROR") {
      return "Network error (request interrupted or unreachable)";
    }
    return name;
  }
  if ("status" in error) {
    return `HTTP ${String((error as { status: unknown }).status)}`;
  }
  return "request failed";
}
/**
 * Extracts the classified GitHub failure (rate limit, 401/402/403, outage, no
 * network) the admin API attaches to blueprint errors, or null for any other
 * error. Callers use it to show the GitHub error screen instead of a toast.
 */
export function githubHealthFromError(error: unknown): GitHubHealth | null {
  if (!error || typeof error !== "object" || !("data" in error)) return null;
  const data = (error as { data?: unknown }).data;
  if (!data || typeof data !== "object" || !("github" in data)) return null;
  const gh = (data as { github?: unknown }).github;
  if (!gh || typeof gh !== "object" || typeof (gh as { state?: unknown }).state !== "string") {
    return null;
  }
  return gh as GitHubHealth;
}
