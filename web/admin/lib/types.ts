export type ServiceInstance = {
  id: string;
  type: "postgres" | "mysql" | "redis" | "rustfs" | "qdrant" | "searxng";
  display_name?: string;
  enabled: boolean;
  credentials: {
    username?: string;
    password?: string;
    database?: string;
    access_key?: string;
    secret_key?: string;
  };
  state?: string;
};

export type StatusResponse = {
  profile?: string;
  tailscale?: boolean;
  reconcileMode?: "simulate" | "cluster";
  serviceStates?: Record<string, string>;
  instances?: ServiceInstance[];
  messages?: string[];
  error?: string;
  machineId?: string;
  backup?: {
    ready?: boolean;
    credentials_configured?: boolean;
    remote_metadata_ready?: boolean;
    implementation_status?: string;
    bucket?: string;
    config_prefix?: string;
    data_prefix?: string;
  };
};

export type BackupTargetResponse = {
  machine_id: string;
  layout: {
    bucket: string;
    config_prefix: string;
    data_prefix: string;
    preflight_key: string;
  };
  remote: {
    endpoint: string;
  };
  credentials_secret: string;
  credentials_configured: boolean;
  remote_metadata_ready: boolean;
  backup_ready: boolean;
  last_test_at?: string;
  last_test_ok?: boolean;
  last_test_message?: string;
  implementation_status: string;
};

export type ConnectionsResponse = {
  instances?: {
    id: string;
    type: string;
    display_name?: string;
    postgres?: string;
    mysql?: string;
    redis?: string;
    s3_http?: string;
    qdrant_http?: string;
    qdrant_grpc?: string;
    searxng_http?: string;
  }[];
  notes?: string;
};

export type CreateInstanceResponse = {
  instance: ServiceInstance;
  credentials: ServiceInstance["credentials"];
};

export type OpenRouterAccountView = {
  id: string;
  label: string;
  enabled: boolean;
  notes?: string;
  has_api_key: boolean;
  upstream_label?: string;
  key_tail?: string;
  upstream_user_id?: string;
  key_fingerprint?: string;
};

export type MailAccountView = {
  id: string;
  label: string;
  enabled: boolean;
  address: string;
  provider?: string;
  has_secret: boolean;
  notes?: string;
};

export type MailConfigResponse = {
  enabled: boolean;
  gateway_token_configured: boolean;
  gateway_public_path: string;
  gateway_direct_path: string;
  gateway_local_base_url?: string;
  accounts: MailAccountView[];
  runtime: {
    enabled: boolean;
    configured: boolean;
    last_account_id?: string;
    enabled_account_count: number;
  };
};

export type OpenRouterConfigResponse = {
  enabled: boolean;
  gateway_token_configured: boolean;
  gateway_public_path: string;
  gateway_direct_path: string;
  gateway_local_base_url?: string;
  accounts: OpenRouterAccountView[];
  runtime: {
    enabled: boolean;
    configured: boolean;
    last_account_id?: string;
    enabled_account_count: number;
  };
};

export type OverviewComponent = {
  key: string;
  kind: "platform" | "instance";
  name: string;
  type?: string;
  instance_id?: string;
  desired: boolean;
  state: string;
  helm_status?: string;
  revision?: number;
  reason?: string;
  pods_ready?: number;
  pods_total?: number;
  release: string;
};

export type OverviewHistoryEvent = {
  at: string;
  key: string;
  kind: string;
  state: string;
  detail?: string;
  helm_status?: string;
};

export type OverviewResponse = {
  generated_at: string;
  reconcileMode: string;
  profile?: string;
  tailscale?: boolean;
  machineId?: string;
  backup_ready?: boolean;
  components: OverviewComponent[];
  summary: {
    running: number;
    stopped: number;
    pending: number;
    degraded: number;
    other: number;
  };
  history: OverviewHistoryEvent[];
  history_hours: number;
};