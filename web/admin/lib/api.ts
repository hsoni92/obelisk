/**
 * Domain types and RTK Query hooks for the Obelisk admin API.
 */
export type {
  BackupTargetResponse,
  ConnectionsResponse,
  CreateInstanceResponse,
  MailAccountView,
  MailConfigResponse,
  OpenRouterAccountView,
  OpenRouterConfigResponse,
  ServiceInstance,
  OverviewResponse,
  StatusResponse,
} from "@/lib/types";

export {
  obeliskApi,
  rtkErrorMessage,
  overviewPollOptions,
  statusPollOptions,
  useCreateInstanceMutation,
  useGetOverviewQuery,
  useGetBackupTargetQuery,
  useGetConnectionsQuery,
  useGetStatusQuery,
  usePatchBackupTargetMutation,
  usePatchInstanceMutation,
  useRunBackupMutation,
  useTestBackupRemoteMutation,
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
} from "@/lib/store/obelisk-api";