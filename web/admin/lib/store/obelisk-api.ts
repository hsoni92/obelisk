import type {
  BackupTargetResponse,
  ConnectionsResponse,
  CreateInstanceResponse,
  MailConfigResponse,
  OpenRouterConfigResponse,
  ServiceInstance,
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
    prepareHeaders: (headers) => {
      headers.set("Accept", "application/json");
      return headers;
    },
  }),
  tagTypes: ["Status", "Connections", "BackupTarget", "Overview", "OpenRouter", "Mail"],
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
    getConnections: builder.query<ConnectionsResponse, void>({
      query: () => "/connections",
      providesTags: ["Connections"],
    }),
    getBackupTarget: builder.query<BackupTargetResponse, void>({
      query: () => "/backup/target",
      providesTags: ["BackupTarget"],
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
          display_name?: string;
          enabled?: boolean;
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
      invalidatesTags: ["BackupTarget", "Status"],
    }),
    runBackup: builder.mutation<{ status: string; message?: string }, void>({
      query: () => ({
        url: "/backup/run",
        method: "POST",
      }),
      invalidatesTags: ["Status"],
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
  useGetConnectionsQuery,
  useGetBackupTargetQuery,
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
} = obeliskApi;

/** Default polling for live cluster status. */
export const statusPollOptions = { pollingInterval: POLL_MS } as const;

/** Overview polls helm/kubectl; keep slightly slower than /status. */
export const overviewPollOptions = { pollingInterval: 8000 } as const;

export function rtkErrorMessage(error: unknown): string {
  if (!error || typeof error !== "object") return "request failed";
  if ("data" in error) {
    const data = (error as { data?: unknown }).data;
    if (typeof data === "string" && data.trim()) return data;
    if (typeof data === "object" && data) {
      if ("message" in data) {
        return String((data as { message: unknown }).message);
      }
      if ("error" in data && (data as { error: unknown }).error === "duplicate_key") {
        const d = data as { message?: string; existing_label?: string; key_tail?: string };
        return (
          d.message ??
          `Duplicate key (already saved as ${d.existing_label ?? "account"}, tail …${d.key_tail ?? ""})`
        );
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