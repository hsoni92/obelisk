package obelisk

// Spec is the declarative desired state (obelisk.yaml).
type Spec struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Machine    Machine      `yaml:"machine,omitempty"`
	Deployment Deployment   `yaml:"deployment"`
	Hardware   Hardware     `yaml:"hardware"`
	Services   ServiceFlags       `yaml:"services"`
	Instances  []ServiceInstance  `yaml:"instances,omitempty"`
	Backup            Backup            `yaml:"backup"`
	Exposure          Exposure          `yaml:"exposure"`
	OpenRouterGateway OpenRouterGateway `yaml:"openrouter_gateway,omitempty" json:"openrouter_gateway,omitempty"`
	MailGateway       MailGateway       `yaml:"mail_gateway,omitempty" json:"mail_gateway,omitempty"`
	Projects          []Project         `yaml:"projects"`
}

type Deployment struct {
	Profile   string          `yaml:"profile"` // vps | local
	Tailscale TailscaleDeploy `yaml:"tailscale"`
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
	Redis    ServiceToggle `yaml:"redis"`
	RustFS   ServiceToggle `yaml:"rustfs"`
	Qdrant   ServiceToggle `yaml:"qdrant"`
	SearXNG  ServiceToggle `yaml:"searxng"`
}

type ServiceToggle struct {
	Enabled bool   `yaml:"enabled"`
	Profile string `yaml:"profile,omitempty"`
}

type Backup struct {
	Remote            RemoteBackup `yaml:"remote"`
	CredentialsSecret string       `yaml:"credentials_secret,omitempty" json:"credentials_secret,omitempty"`
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

type Project struct {
	Name        string `yaml:"name"`
	PostgresDB  string `yaml:"postgres_db"`
	RedisDB     int    `yaml:"redis_db"`
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
			Tailscale: TailscaleDeploy{
				Enabled: false,
			},
		},
		Hardware: Hardware{Tier: "floor"},
		Services: ServiceFlags{
			Postgres: ServiceToggle{Enabled: false},
			MySQL:    ServiceToggle{Enabled: false},
			Redis:    ServiceToggle{Enabled: false},
			RustFS:   ServiceToggle{Enabled: false},
			Qdrant:   ServiceToggle{Enabled: false},
			SearXNG:  ServiceToggle{Enabled: false},
		},
		Exposure: Exposure{
			Funnel: FunnelExposure{
				RustFSS3: FunnelRoute{Enabled: false},
			},
		},
		Projects: []Project{},
		Backup: Backup{
			CredentialsSecret: DefaultBackupCredentialsSecret,
		},
		OpenRouterGateway: OpenRouterGateway{Enabled: false},
		MailGateway:       MailGateway{Enabled: false},
	}
}