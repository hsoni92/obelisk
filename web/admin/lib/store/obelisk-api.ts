import type {
  BackupConfigResponse,
  BackupTargetResponse,
  BlueprintAuthorizeResponse,
  BlueprintStatus,
  ConnectionsResponse,
  CreateInstanceResponse,
  MailConfigResponse,
  OpenRouterConfigResponse,
  RestoreChainState,
  ServiceInstance,
  ActivityHistoryResponse,
  OverviewResponse,
  StatusResponse,
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
  tagTypes: ["Status", "Connections", "BackupTarget", "BackupConfig", "Overview", "OpenRouter", "Mail", "Blueprint", "Restore"],
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
    deleteBackupDestination: builder.mutation<BackupConfigResponse, { id: string }>({
      query: ({ id }) => ({
        url: `/backup/destinations/${encodeURIComponent(id)}`,
        method: "DELETE",
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
    createInstance: builder.mutation<
      CreateInstanceResponse,
      { type: string; display_name: string }
    >({
      query: (body) => ({
        url: "/instances",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Status", "Connections", "Overview"],
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
      invalidatesTags: ["Status", "BackupConfig"],
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
    getBlueprintStatus: builder.query<BlueprintStatus, void>({
      query: () => "/blueprint/status",
      providesTags: ["Blueprint"],
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
    }),
    unlockBlueprint: builder.mutation<BlueprintStatus, { passphrase: string }>({
      query: (body) => ({
        url: "/blueprint/unlock",
        method: "POST",
        body,
      }),
      invalidatesTags: ["Blueprint"],
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
      ],
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
  useGetOverviewQuery,
  useGetActivityHistoryQuery,
  useGetConnectionsQuery,
  useGetBackupTargetQuery,
  useGetBackupConfigQuery,
  usePatchBackupPolicyMutation,
  usePostBackupDestinationMutation,
  usePatchBackupDestinationMutation,
  useDeleteBackupDestinationMutation,
  useTestBackupDestinationMutation,
  useTestAllBackupDestinationsMutation,
  useCreateInstanceMutation,
  usePatchInstanceMutation,
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
  usePatchMailConfigMutation,
  usePostMailAccountMutation,
  usePatchMailAccountMutation,
  useDeleteMailAccountMutation,
  useTestMailMutation,
  useGetBlueprintStatusQuery,
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
} = obeliskApi;

/** Default polling for live cluster status. */
export const statusPollOptions = { pollingInterval: POLL_MS } as const;

/** Overview polls helm/kubectl; keep slightly slower than /status. */
export const overviewPollOptions = { pollingInterval: 8000 } as const;

/** Activity history is file-backed only — safe to poll while the drawer is open. */
export const activityHistoryPollOptions = { pollingInterval: 8000 } as const;

export function rtkErrorMessage(error: unknown): string {
  if (!error || typeof error !== "object") return "request failed";
  if ("data" in error) {
    const data = (error as { data?: unknown }).data;
    if (typeof data === "string" && data.trim()) return data;
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
    return (error as { error: string }).error;
  }
  if ("status" in error) {
    return `HTTP ${String((error as { status: unknown }).status)}`;
  }
  return "request failed";
}