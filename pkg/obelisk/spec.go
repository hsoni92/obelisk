package obelisk

// Spec is the declarative desired state (obelisk.yaml).
type Spec struct {
	APIVersion        string            `yaml:"apiVersion"`
	Kind              string            `yaml:"kind"`
	Machine           Machine           `yaml:"machine,omitempty"`
	Deployment        Deployment        `yaml:"deployment"`
	Hardware          Hardware          `yaml:"hardware"`
	Services          ServiceFlags      `yaml:"services"`
	Instances         []ServiceInstance `yaml:"instances,omitempty"`
	Backup            Backup            `yaml:"backup"`
	Exposure          Exposure          `yaml:"exposure"`
	OpenRouterGateway OpenRouterGateway `yaml:"openrouter_gateway,omitempty" json:"openrouter_gateway,omitempty"`
	MailGateway       MailGateway       `yaml:"mail_gateway,omitempty" json:"mail_gateway,omitempty"`
	Blueprint         Blueprint         `yaml:"blueprint,omitempty" json:"blueprint,omitempty"`
	Projects          []Project         `yaml:"projects"`
	PeerGateway       PeerGateway       `yaml:"peer_gateway,omitempty" json:"peer_gateway,omitempty"`
	Peers             []PeerAdmission   `yaml:"peers,omitempty" json:"peers,omitempty"`
}

type Deployment struct {
	Profile   string          `yaml:"profile"`             // vps | local
	Role      string          `yaml:"role,omitempty"`      // local | gateway (default local)
	Tailscale TailscaleDeploy `yaml:"tailscale"`
}

// PeerGateway configures tailnet federation (gateway aggregates locals).
type PeerGateway struct {
	AdvertiseURL       string `yaml:"advertise_url,omitempty"`         // local: gateway advertise endpoint
	GatewayEdgeHost    string `yaml:"gateway_edge_host,omitempty"`     // local: gateway public host / edge zone (usually learned from advertise)
	AutoAdmit          *bool  `yaml:"auto_admit,omitempty"`            // gateway: default true when omitted (OQ2)
	ExposeNativePublic *bool  `yaml:"expose_native_public,omitempty"`  // gateway: default true (OQ4)
	TLSMode            string `yaml:"tls_mode,omitempty"`              // v1: passthrough only (OQ1)
}

// PeerAdmission is a persisted peer row on the gateway.
type PeerAdmission struct {
	ID           string `yaml:"id" json:"id"`
	MachineID    string `yaml:"machine_id,omitempty" json:"machine_id,omitempty"`
	DisplayName  string `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	TailnetHost  string `yaml:"tailnet_host,omitempty" json:"tailnet_host,omitempty"`
	Admitted     bool   `yaml:"admitted" json:"admitted"`
}

type TailscaleDeploy struct {
	Enabled bool `yaml:"enabled"`
}

type Hardware struct {
	Tier string `yaml:"tier"` // floor | comfort
}

type ServiceFlags struct {
	Postgres ServiceToggle `yaml:"postgres"`
	MySQL    ServiceToggle `yaml:"mysql"`
	MongoDB  ServiceToggle `yaml:"mongodb"`
	Redis    ServiceToggle `yaml:"redis"`
	RustFS   ServiceToggle `yaml:"rustfs"`
	Qdrant   ServiceToggle `yaml:"qdrant"`
	SearXNG  ServiceToggle `yaml:"searxng"`
	Redpanda ServiceToggle `yaml:"redpanda"`
	Unimeter ServiceToggle `yaml:"unimeter"`
}

type ServiceToggle struct {
	Enabled bool   `yaml:"enabled"`
	Profile string `yaml:"profile,omitempty"`
}

type Backup struct {
	Policy            BackupPolicy        `yaml:"policy,omitempty" json:"policy,omitempty"`
	Destinations      []BackupDestination `yaml:"destinations,omitempty" json:"destinations,omitempty"`
	Remote            RemoteBackup        `yaml:"remote"`
	CredentialsSecret string              `yaml:"credentials_secret,omitempty" json:"credentials_secret,omitempty"`
}

type RemoteBackup struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	// Deprecated: v1 uses machine.id as the bucket name. Legacy fields are ignored after migration.
	DataBucket   string `yaml:"data_bucket,omitempty" json:"data_bucket,omitempty"`
	ConfigBucket string `yaml:"config_bucket,omitempty" json:"config_bucket,omitempty"`
	Bucket       string `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	ObjectPrefix string `yaml:"object_prefix,omitempty" json:"object_prefix,omitempty"`
	Prefix       string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
}

// DefaultBackupCredentialsSecret is the Kubernetes Secret name for S3/R2 upload keys.
const DefaultBackupCredentialsSecret = "obelisk-backup-remote"

type Exposure struct {
	Funnel FunnelExposure `yaml:"funnel"`
}

type FunnelExposure struct {
	RustFSS3 FunnelRoute `yaml:"rustfs_s3"`
}

type FunnelRoute struct {
	Enabled bool `yaml:"enabled"`
}

// Project groups service instances for connection discovery (GET /discover/v1/projects/{id}).
type Project struct {
	ID               string   `yaml:"id" json:"id"`
	Name             string   `yaml:"name" json:"name"`
	InstanceIDs      []string `yaml:"instance_ids,omitempty" json:"instance_ids,omitempty"`
	DiscoveryEnabled bool     `yaml:"discovery_enabled,omitempty" json:"discovery_enabled,omitempty"`
	// Deprecated: ignored after migration; was never wired to the UI.
	PostgresDB string `yaml:"postgres_db,omitempty" json:"-"`
	RedisDB    int    `yaml:"redis_db,omitempty" json:"-"`
}

func DefaultSpec(profile string) Spec {
	if profile != "local" {
		profile = "vps"
	}
	return Spec{
		APIVersion: "obelisk.dev/v1",
		Kind:       "ObeliskCluster",
		Deployment: Deployment{
			Profile: profile,
			Role:    DeploymentRoleLocal,
			Tailscale: TailscaleDeploy{
				Enabled: false,
			},
		},
		Hardware: Hardware{Tier: "floor"},
		Services: ServiceFlags{
			Postgres: ServiceToggle{Enabled: false},
			MySQL:    ServiceToggle{Enabled: false},
			MongoDB:  ServiceToggle{Enabled: false},
			Redis:    ServiceToggle{Enabled: false},
			RustFS:   ServiceToggle{Enabled: false},
			Qdrant:   ServiceToggle{Enabled: false},
			SearXNG:  ServiceToggle{Enabled: false},
			Redpanda: ServiceToggle{Enabled: false},
			Unimeter: ServiceToggle{Enabled: false},
		},
		Exposure: Exposure{
			Funnel: FunnelExposure{
				RustFSS3: FunnelRoute{Enabled: false},
			},
		},
		Projects: []Project{},
		Backup: Backup{
			CredentialsSecret: DefaultBackupCredentialsSecret,
			Policy:            DefaultBackupPolicy(),
		},
		OpenRouterGateway: OpenRouterGateway{Enabled: false},
		MailGateway:       MailGateway{Enabled: false},
	}
}
