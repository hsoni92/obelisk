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

export type BackupLineageWitness = {
  schedule: string;
  replication_factor: number;
  data_retention_days: number;
};

export type BackupLineageNode = {
  id: string;
  parent_id?: string;
  created_at: string;
  trigger: string;
  object_key: string;
  manifest_key: string;
  size: number;
  witness: BackupLineageWitness;
  active: boolean;
};

export type InstanceBackupLineageResponse = {
  service_type: string;
  instance_id: string;
  lineage_id?: string;
  display_name?: string;
  orphan?: boolean;
  clone?: boolean;
  clone_source_key?: string;
  active_backup_id?: string;
  source_backup_id?: string;
  nodes: BackupLineageNode[];
};

export type CloneBackupResponse = {
  status: string;
  message?: string;
  scope?: string;
  source_key?: string;
  service_type?: string;
  instance_id?: string;
  lineage_id?: string;
  node_count?: number;
  uploaded_destinations?: number;
};

export type BackupCatalogRow = {
  service_type: string;
  instance_id: string;
  display_name?: string;
  instance_live: boolean;
  backup_enabled: boolean;
  orphan: boolean;
  state?: string;
  backup_count: number;
  active_backup_id?: string;
  active_backup_at?: string;
  active_backup_bytes?: number;
  last_upload_at?: string;
  lineage_id?: string;
  clone?: boolean;
  clone_source_key?: string;
  replication_factor?: number;
  replication_overridden?: boolean;
  replication_from_witness?: boolean;
};

export type BackupCatalogResponse = {
  generated_at: string;
  global_replication_factor: number;
  enabled_destination_count: number;
  ready_destination_count: number;
  replication_ready: boolean;
  rows: BackupCatalogRow[];
};

export type InstanceDeploymentView = {
  release: string;
  namespace: string;
  backup: InstanceBackupView;
};

export type ServiceInstance = {
  id: string;
  type:
    | "postgres"
    | "mysql"
    | "mongodb"
    | "redis"
    | "rustfs"
    | "qdrant"
    | "searxng"
    | "redpanda"
    | "unimeter"
    | "bunqueue"
    | "browserless";
  display_name?: string;
  enabled: boolean;
  state?: string;
  origin?: "local" | "federated";
  peer_id?: string;
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

export type ProjectView = {
  id: string;
  name: string;
  instance_ids?: string[];
  discovery_enabled: boolean;
  discovery_configured: boolean;
  /** Path on the admin/Kong HTTP edge (remote clients). */
  discovery_url?: string;
  /** Same path with `?audience=in_cluster` (append to your public or Kong base). */
  discovery_url_in_cluster?: string;
  /** Full URL for pods: Kong Service + in-cluster audience. */
  discovery_kong_in_cluster_url?: string;
};

export type TailscaleLiveStatus = {
  installed?: boolean;
  connected?: boolean;
  backend_ok?: boolean;
  backend_state?: string;
  auth_url?: string;
  tailnet?: string;
  dns_name?: string;
  ips?: string[];
  socket_path?: string;
  error?: string;
};

export type TailscaleStatusResponse = {
  status: TailscaleLiveStatus;
  desired_enabled?: boolean;
  deployment_role?: string;
  deployment_profile?: string;
  /** Present when deployment_role is gateway — locals POST catalog here. */
  gateway_advertise_endpoint?: string;
  /** Same path on the gateway Tailscale IP/DNS (use if public URL fails). */
  gateway_advertise_endpoint_tailnet?: string;
  error?: string;
};

export type FederationPeerRecord = {
  id: string;
  machine_id?: string;
  tailnet_host?: string;
  admitted: boolean;
  last_seen?: string;
  reachable?: boolean;
  instances?: Record<
    string,
    { type: string; id: string; enabled: boolean }
  >;
};

export type FederationAdvertiseStatus = {
  ok: boolean;
  last_ok_at?: string;
  last_attempt_at?: string;
  last_error?: string;
};

export type FederationResponse = {
  role: string;
  profile: string;
  tailscale_enabled: boolean;
  is_gateway: boolean;
  advertise_url?: string;
  tailnet_host?: string;
  auto_admit?: boolean;
  gateway_advertise_endpoint?: string;
  gateway_advertise_endpoint_tailnet?: string;
  peers?: FederationPeerRecord[];
  advertise_status?: FederationAdvertiseStatus;
  advertise_configured?: boolean;
};

export type StatusResponse = {
  profile?: string;
  role?: string;
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
  is_primary?: boolean;
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

export type ConnectionBlock = {
  use?: string;
  host?: string;
  postgres?: string;
  mysql?: string;
  mongodb?: string;
  redis?: string;
  s3_http?: string;
  qdrant_http?: string;
  qdrant_grpc?: string;
  searxng_http?: string;
  kafka_bootstrap?: string;
  schema_registry_http?: string;
  unimeter_api?: string;
  unimeter_health_http?: string;
  bunqueue_tcp?: string;
  bunqueue_health_http?: string;
  browserless_http?: string;
  browserless_ws?: string;
  databases?: { name: string; url: string }[];
};

export type ConnectionAudience = "edge" | "in_cluster";

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
    kafka_bootstrap?: string;
    schema_registry_http?: string;
    unimeter_api?: string;
    unimeter_health_http?: string;
  bunqueue_tcp?: string;
  bunqueue_health_http?: string;
    browserless_http?: string;
    browserless_ws?: string;
    in_cluster?: ConnectionBlock;
  }[];
  notes?: string;
  in_cluster_notes?: string;
};

export type CreateInstanceFromBackupResult = {
  source_type: string;
  source_instance_id: string;
  backup_id: string;
  restore_ok: boolean;
  restore_error?: string;
};

export type CreateInstanceResponse = {
  instance: ServiceInstance;
  credentials: ServiceInstance["credentials"];
  use?: string;
  from_backup?: CreateInstanceFromBackupResult;
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
  default_upstream_branch?: string;
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
  /** Whether GitHub can be used right now; blocking states replace sign-in/onboarding. */
  github_api?: GitHubHealth;
  /** The blueprint repo could not be read, so bundle_present/needs_onboarding are unknown. */
  remote_check_failed?: boolean;
};

export type GitHubHealthState =
  | "ok"
  | "rate_limited"
  | "forbidden"
  | "unauthorized"
  | "payment_required"
  | "unavailable"
  | "unreachable"
  | "error";

export type GitHubHealth = {
  state: GitHubHealthState;
  /** Sign-in, onboarding and blueprint sync cannot work while this is set. */
  blocking: boolean;
  http_status?: number;
  title: string;
  message?: string;
  hint?: string;
  steps?: string[];
  reset_at?: string;
  retry_after_seconds?: number;
  limit?: number;
  remaining?: number;
  sso_url?: string;
  request_id?: string;
  token_present: boolean;
  latency_ms?: number;
  checked_at: string;
};

export type SystemCheckState = "ok" | "warn" | "fail" | "off";

export type SystemCheck = {
  id: string;
  group: "github" | "platform";
  name: string;
  state: SystemCheckState;
  summary: string;
  detail?: string;
  hint?: string;
  latency_ms?: number;
  blocking?: boolean;
  github?: GitHubHealth;
};

export type SystemHealthReport = {
  checked_at: string;
  overall: SystemCheckState;
  checks: SystemCheck[];
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
  running_since?: string;
  pod_phase?: string;
  restart_count?: number;
  last_termination_reason?: string;
  pod_ready?: boolean;
  waiting_reason?: string;
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
  role?: string;
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
  warnings?: string[];
};
export type ConnectivityState =
  | "connected"
  | "degraded"
  | "offline"
  | "never"
  | "no_data"
  | "unknown";

export type ConnectivityPeer = {
  id: string;
  machine_id?: string;
  tailnet_host?: string;
  duplicate_machine_id: boolean;
  last_seen?: string;
  heartbeat_fresh: boolean;
  probe_known: boolean;
  probe_reachable: boolean;
  probe_checked_at?: string;
  probe_error?: string;
  state: ConnectivityState;
  state_since?: string;
  state_since_capped: boolean;
  observed_seconds: number;
  strip: ConnectivityState[];
};

export type ConnectivityResponse = {
  role: string;
  is_gateway: boolean;
  peer_view_enabled: boolean;
  window_start?: string;
  window_hours: number;
  bin_minutes: number;
  gateway_alive_at?: string;
  monitoring_since?: string;
  heartbeat_stale_seconds: number;
  probe_interval_seconds: number;
  counts: Partial<Record<ConnectivityState, number>>;
  peers: ConnectivityPeer[];
};

export type UpdateStatus = {
  installed_build: number;
  latest_build?: number;
  source?: "release" | "dev";
  update_available: boolean;
  checkable: boolean;
  checked_at: string;
  error?: string;
  command: string;
  apply?: UpdateApply;
};

export type UpdateApply = {
  phase: "running" | "succeeded" | "failed";
  target_build?: number;
  started_at?: string;
  log_tail?: string;
};
