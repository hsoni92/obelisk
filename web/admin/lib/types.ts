export type InstanceBackupPolicyView = {
  schedule: string;
  replication_factor: number;
  data_retention_days: number;
  overridden: boolean;
};

export type InstanceBackupView = {
  supported: boolean;
  enabled: boolean;
  method?: string;
  object_prefix?: string;
  policy?: InstanceBackupPolicyView;
};

export type InstanceDeploymentView = {
  release: string;
  namespace: string;
  backup: InstanceBackupView;
};

export type ServiceInstance = {
  id: string;
  type: "postgres" | "mysql" | "mongodb" | "redis" | "rustfs" | "qdrant" | "searxng";
  display_name?: string;
  enabled: boolean;
  state?: string;
  deployment?: InstanceDeploymentView;
  backup?: {
    enabled?: boolean;
    policy_override?: {
      schedule?: string;
      replication_factor?: number;
      data_retention_days?: number;
    };
  };
  credentials: {
    username?: string;
    password?: string;
    database?: string;
    access_key?: string;
    secret_key?: string;
  };
  /** Extra logical DBs (postgres/mysql/mongodb). Bootstrap DB is always `default`. */
  databases?: string[];
};

export type InstanceDatabasesResponse = {
  bootstrap_database: string;
  additional_databases: string[];
  connections: { name: string; url: string }[];
};

export type StatusResponse = {
  profile?: string;
  tailscale?: boolean;
  reconcileMode?: "off" | "cluster";
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

export type BackupDestinationView = {
  id: string;
  label: string;
  provider?: string;
  endpoint: string;
  bucket: string;
  enabled: boolean;
  credentials_configured: boolean;
  ready: boolean;
  status: string;
  last_test_at?: string;
  last_test_ok?: boolean;
  last_test_message?: string;
  last_upload_at?: string;
  last_inventory_at?: string;
  bytes?: number;
  objects?: number;
  last_error?: string;
  schedule_lag?: string;
};

export type BackupConfigResponse = {
  machine_id: string;
  layout: BackupTargetResponse["layout"];
  scope?: {
    data_service_types: string[];
  };
  policy: {
    schedule: string;
    replication_factor: number;
    data_retention_days: number;
    config_keep_count: number;
    include_config: boolean;
  };
  destinations: BackupDestinationView[];
  backup_ready: boolean;
  replication_ready: boolean;
  ready_destination_count: number;
  enabled_destination_count: number;
  implementation_status: string;
  last_run_at?: string;
  /** "<type>/<id>" → ISO time of that instance's last data upload. */
  last_backup_by_instance?: Record<string, string>;
  credentials_secret: string;
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

export type DiagnosticCheck = {
  name: string;
  ok: boolean;
  detail?: string;
};

export type DiagnosticInstanceResult = {
  type: string;
  id: string;
  display_name?: string;
  cluster_state: string;
  ok: boolean;
  skipped?: boolean;
  skipped_reason?: string;
  duration_ms: number;
  summary: string;
  public_host?: string;
  checks?: DiagnosticCheck[];
};

export type DiagnosticsRunResponse = {
  started_at: string;
  results: DiagnosticInstanceResult[];
};

export type ConnectionsResponse = {
  instances?: {
    id: string;
    type: string;
    display_name?: string;
    route_id?: string;
    public_path?: string;
    /** Single copy-paste credential block for this service (includes secrets). */
    use?: string;
    postgres?: string;
    mysql?: string;
    mongodb?: string;
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
  use?: string;
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

export type BlueprintStatus = {
  authenticated: boolean;
  login?: string;
  client_configured: boolean;
  repository?: string;
  branch?: string;
  generation: number;
  remote_generation?: number;
  last_sync_at?: string;
  last_commit_sha?: string;
  sync_status: string;
  crypto_unlocked: boolean;
  import_required: boolean;
  last_push_error?: string;
  machine_id?: string;
  remote_machine_id?: string;
  machine_id_mismatch: boolean;
  bundle_present: boolean;
  needs_onboarding: boolean;
  discovered_repository?: string;
  key_id?: string;
};

export type BlueprintAuthorizeResponse = {
  flow: "code" | "device";
  authorize_url?: string;
  state?: string;
  user_code?: string;
  verification_uri?: string;
  device_code?: string;
  interval?: number;
  expires_in?: number;
};

export type RestoreChainState = {
  phase: string;
  settings_restored: boolean;
  data_restored: boolean;
  data_objects: number;
  error?: string;
  started_at?: string;
  finished_at?: string;
  phases?: string[];
  running: boolean;
};

export type OverviewHistoryEvent = {
  at: string;
  key: string;
  kind: string;
  state: string;
  detail?: string;
  helm_status?: string;
};

export type ActivityHistoryResponse = {
  history: OverviewHistoryEvent[];
  history_hours: number;
};

export type VitalsPoint = {
  at: string;
  percent: number;
};

export type VitalsMeter = {
  used_bytes?: number;
  total_bytes?: number;
  percent: number;
  known: boolean;
  observed_at?: string;
  not_running?: boolean;
  volume_labels?: string[];
  series?: VitalsPoint[];
};

export type VitalsRow = {
  key: string;
  name: string;
  kind: string;
  type?: string;
  running: boolean;
  rank_driver?: string;
  memory: VitalsMeter;
  cpu: VitalsMeter;
  disk: VitalsMeter;
};

export type VitalsSnapshot = {
  collected_at: string;
  stale: boolean;
  collecting: boolean;
  scarce_resource?: string;
  machine: {
    memory_used_bytes: number;
    memory_total_bytes: number;
    disk_used_bytes: number;
    disk_total_bytes: number;
    cpu_percent: number;
    memory_series?: VitalsPoint[];
    disk_series?: VitalsPoint[];
    cpu_series?: VitalsPoint[];
    pressure_memory?: boolean;
    pressure_disk?: boolean;
    pressure_pid?: boolean;
  };
  unaccounted?: {
    memory_bytes: number;
    disk_bytes: number;
    cpu_percent: number;
  };
  rows: VitalsRow[];
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
  vitals?: VitalsSnapshot;
};