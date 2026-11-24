package obelisk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// InstanceCredentials are persisted in obelisk.yaml (v1 dev; production may move to sealed secrets).
type InstanceCredentials struct {
	Username  string `yaml:"username,omitempty" json:"username,omitempty"`
	Password  string `yaml:"password,omitempty" json:"password,omitempty"`
	Database  string `yaml:"database,omitempty" json:"database,omitempty"`
	AccessKey string `yaml:"access_key,omitempty" json:"access_key,omitempty"`
	SecretKey string `yaml:"secret_key,omitempty" json:"secret_key,omitempty"`
}

// ServiceInstance is one deployable data-service workload.
type ServiceInstance struct {
	// ID is the immutable route slug (e.g. calm-otter) used in edge hostnames and Helm release names.
	ID          string                 `yaml:"id" json:"id"`
	Type        string                 `yaml:"type" json:"type"` // postgres | mysql | mongodb | redis | rustfs | qdrant | searxng | redpanda | unimeter | bunqueue | browserless
	DisplayName string                 `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Backup      InstanceBackupSettings `yaml:"backup,omitempty" json:"backup,omitempty"`
	Credentials InstanceCredentials    `yaml:"credentials" json:"credentials"`
	// Databases lists extra logical DBs beyond BootstrapDatabaseName (relational engines only).
	Databases []string `yaml:"databases,omitempty" json:"databases,omitempty"`
}

func (i ServiceInstance) ReleaseName() string {
	if i.ID == "default" {
		return "obelisk-" + i.Type
	}
	return fmt.Sprintf("obelisk-%s-%s", i.Type, i.ID)
}

func (i ServiceInstance) StateKey() string {
	return i.Type + "/" + i.ID
}

// DefaultCredentialsForType returns dev defaults for a new instance.
func DefaultCredentialsForType(typ string) InstanceCredentials {
	switch typ {
	case "postgres", "mysql", "mongodb":
		return InstanceCredentials{
			Username: "obelisk",
			Password: randomDevSecret(16),
			Database: BootstrapDatabaseName,
		}
	case "redis":
		return InstanceCredentials{
			Password: randomDevSecret(16),
		}
	case "rustfs":
		return InstanceCredentials{
			AccessKey: randomDevSecret(16),
			SecretKey: randomDevSecret(24),
		}
	case "qdrant":
		return InstanceCredentials{
			AccessKey: randomDevSecret(32),
		}
	case "searxng":
		return InstanceCredentials{
			SecretKey: randomDevSecret(32),
		}
	case "browserless":
		return InstanceCredentials{
			AccessKey: randomDevSecret(32),
		}
	case "redpanda":
		return InstanceCredentials{
			Username: "obelisk",
			Password: randomDevSecret(16),
		}
	default:
		return InstanceCredentials{}
	}
}

func randomDevSecret(n int) string {
	b := make([]byte, n/2+1)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// NewServiceInstance creates an instance with defaults and a caller-supplied id.
func NewServiceInstance(typ, id, displayName string) (ServiceInstance, error) {
	if err := ValidateServiceType(typ); err != nil {
		return ServiceInstance{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" || !instanceIDValid(id) {
		return ServiceInstance{}, fmt.Errorf("invalid instance id")
	}
	inst := ServiceInstance{
		ID:          id,
		Type:        typ,
		DisplayName: displayNameOr(id, displayName),
		Enabled:     true,
		Backup:      DefaultInstanceBackupSettings(typ),
		Credentials: DefaultCredentialsForType(typ),
	}
	if IsRelationalEngine(typ) {
		inst.Credentials.Database = BootstrapDatabaseName
	}
	return inst, nil
}

func displayNameOr(id, name string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return id
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// SlugID normalizes a human name to a k8s-safe id segment.
func SlugID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}

func ValidateServiceType(typ string) error {
	switch typ {
	case "postgres", "mysql", "mongodb", "redis", "rustfs", "qdrant", "searxng", "redpanda", "unimeter", "bunqueue", "browserless":
		return nil
	default:
		return fmt.Errorf("type must be postgres, mysql, mongodb, redis, rustfs, qdrant, searxng, redpanda, unimeter, bunqueue, or browserless")
	}
}

// ResolvedDBCreds returns username, password, and database for logical backup tooling.
// Empty username/database are filled with product defaults so backups match Helm after restore.
func ResolvedDBCreds(inst ServiceInstance) (user, password, database string) {
	password = inst.Credentials.Password
	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		user = strings.TrimSpace(inst.Credentials.Username)
		if user == "" {
			user = "obelisk"
		}
		database = strings.TrimSpace(inst.Credentials.Database)
		if database == "" {
			database = BootstrapDatabaseName
		}
	default:
		user = strings.TrimSpace(inst.Credentials.Username)
		database = strings.TrimSpace(inst.Credentials.Database)
	}
	return user, password, database
}

func normalizeInstanceCredentials(inst *ServiceInstance) {
	if inst == nil {
		return
	}
	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		if strings.TrimSpace(inst.Credentials.Username) == "" {
			inst.Credentials.Username = "obelisk"
		}
		if strings.TrimSpace(inst.Credentials.Database) == "" {
			inst.Credentials.Database = BootstrapDatabaseName
		}
	case "redpanda":
		if strings.TrimSpace(inst.Credentials.Username) == "" {
			inst.Credentials.Username = "obelisk"
		}
	}
}

// NormalizeInstances migrates legacy service flags into instances[] and keeps flags in sync.
func NormalizeInstances(spec *Spec) {
	if len(spec.Instances) == 0 {
		if spec.Services.Postgres.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("postgres", "default", spec))
		}
		if spec.Services.MySQL.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("mysql", "default", spec))
		}
		if spec.Services.MongoDB.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("mongodb", "default", spec))
		}
		if spec.Services.Redis.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("redis", "default", spec))
		}
		if spec.Services.RustFS.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("rustfs", "default", spec))
		}
		if spec.Services.Qdrant.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("qdrant", "default", spec))
		}
		if spec.Services.SearXNG.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("searxng", "default", spec))
		}
		if spec.Services.Redpanda.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("redpanda", "default", spec))
		}
		if spec.Services.Unimeter.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("unimeter", "default", spec))
		}
		if spec.Services.Bunqueue.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("bunqueue", "default", spec))
		}
		if spec.Services.Browserless.Enabled {
			spec.Instances = append(spec.Instances, legacyInstance("browserless", "default", spec))
		}
	}
	for i := range spec.Instances {
		normalizeInstanceCredentials(&spec.Instances[i])
	}
	repairDuplicateInstanceIDs(spec)
	remapCanonicalDefaultIDs(spec)
	SyncLegacyServiceFlags(spec)
}

// remapCanonicalDefaultIDs restores id "default" when display name says so and the
// slot is free for that type. Repairs configs that lost release names after import.
func remapCanonicalDefaultIDs(spec *Spec) {
	for i := range spec.Instances {
		inst := &spec.Instances[i]
		if inst.ID == LegacyDefaultInstanceID {
			continue
		}
		name := strings.TrimSpace(strings.ToLower(inst.DisplayName))
		if name != "default" && name != LegacyDefaultInstanceID {
			continue
		}
		if _, _, taken := FindInstanceByTypeAndID(spec, inst.Type, LegacyDefaultInstanceID); taken {
			continue
		}
		inst.ID = LegacyDefaultInstanceID
	}
}

// repairDuplicateInstanceIDs reassigns ids when the same type reuses an instance id.
// Cross-type id reuse (e.g. postgres/default + redis/default) is intentional: release
// names are per-type and must stay stable across blueprint import.
func repairDuplicateInstanceIDs(spec *Spec) {
	seen := map[string]struct{}{}
	for i := range spec.Instances {
		inst := &spec.Instances[i]
		key := inst.Type + "\x00" + inst.ID
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			continue
		}
		prev := inst.ID
		inst.ID = AllocateInstanceIDForType(spec, inst.Type)
		seen[inst.Type+"\x00"+inst.ID] = struct{}{}
		if inst.DisplayName == prev || inst.DisplayName == LegacyDefaultInstanceID {
			inst.DisplayName = displayNameOr(inst.ID, inst.DisplayName)
		}
	}
}

func legacyInstance(typ, id string, spec *Spec) ServiceInstance {
	c := DefaultCredentialsForType(typ)
	// preserve chart dev passwords when migrating existing clusters
	switch typ {
	case "postgres", "mysql", "mongodb":
		c.Password = "changeme-in-ui"
		c.Username = "obelisk"
		c.Database = BootstrapDatabaseName
	case "rustfs":
		c.AccessKey = "obelisk"
		c.SecretKey = "changeme-rustfs-dev"
	case "qdrant":
		c.AccessKey = "changeme-qdrant-dev"
	case "searxng":
		c.SecretKey = "changeme-searxng-dev"
	case "redpanda":
		c.Username = "obelisk"
		c.Password = "changeme-in-ui"
	}
	enabled := false
	switch typ {
	case "postgres":
		enabled = spec.Services.Postgres.Enabled
	case "mysql":
		enabled = spec.Services.MySQL.Enabled
	case "mongodb":
		enabled = spec.Services.MongoDB.Enabled
	case "redis":
		enabled = spec.Services.Redis.Enabled
	case "rustfs":
		enabled = spec.Services.RustFS.Enabled
	case "qdrant":
		enabled = spec.Services.Qdrant.Enabled
	case "searxng":
		enabled = spec.Services.SearXNG.Enabled
	case "redpanda":
		enabled = spec.Services.Redpanda.Enabled
	case "unimeter":
		enabled = spec.Services.Unimeter.Enabled
	case "bunqueue":
		enabled = spec.Services.Bunqueue.Enabled
	case "browserless":
		enabled = spec.Services.Browserless.Enabled
	}
	return ServiceInstance{
		ID:          id,
		Type:        typ,
		DisplayName: id,
		Enabled:     enabled,
		Backup:      DefaultInstanceBackupSettings(typ),
		Credentials: c,
	}
}

// SyncLegacyServiceFlags keeps deprecated boolean flags aligned with instances[].
func SyncLegacyServiceFlags(spec *Spec) {
	spec.Services.Postgres.Enabled = anyInstanceEnabled(spec, "postgres")
	spec.Services.MySQL.Enabled = anyInstanceEnabled(spec, "mysql")
	spec.Services.MongoDB.Enabled = anyInstanceEnabled(spec, "mongodb")
	spec.Services.Redis.Enabled = anyInstanceEnabled(spec, "redis")
	spec.Services.RustFS.Enabled = anyInstanceEnabled(spec, "rustfs")
	spec.Services.Qdrant.Enabled = anyInstanceEnabled(spec, "qdrant")
	spec.Services.SearXNG.Enabled = anyInstanceEnabled(spec, "searxng")
	spec.Services.Redpanda.Enabled = anyInstanceEnabled(spec, "redpanda")
	spec.Services.Unimeter.Enabled = anyInstanceEnabled(spec, "unimeter")
	spec.Services.Bunqueue.Enabled = anyInstanceEnabled(spec, "bunqueue")
	spec.Services.Browserless.Enabled = anyInstanceEnabled(spec, "browserless")
}

func anyInstanceEnabled(spec *Spec, typ string) bool {
	for _, i := range spec.Instances {
		if i.Type == typ && i.Enabled {
			return true
		}
	}
	return false
}

func FindInstance(spec *Spec, id string) (ServiceInstance, int, bool) {
	for i, inst := range spec.Instances {
		if inst.ID == id {
			return inst, i, true
		}
	}
	return ServiceInstance{}, -1, false
}

// FindInstanceByTypeAndID locates an instance when ids are not globally unique (e.g. default postgres + default redis).
func FindInstanceByTypeAndID(spec *Spec, typ, id string) (ServiceInstance, int, bool) {
	typ = strings.TrimSpace(typ)
	id = strings.TrimSpace(id)
	for i, inst := range spec.Instances {
		if inst.Type == typ && inst.ID == id {
			return inst, i, true
		}
	}
	return ServiceInstance{}, -1, false
}

// RemoveInstanceByTypeAndID deletes an instance row from spec (not scale-to-zero).
func RemoveInstanceByTypeAndID(spec *Spec, typ, id string) (ServiceInstance, bool) {
	inst, idx, ok := FindInstanceByTypeAndID(spec, typ, id)
	if !ok {
		return ServiceInstance{}, false
	}
	spec.Instances = append(spec.Instances[:idx], spec.Instances[idx+1:]...)
	return inst, true
}

func UniqueInstanceID(spec *Spec, base string) string {
	if _, _, ok := FindInstance(spec, base); !ok {
		return base
	}
	for n := 2; n < 1000; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if _, _, ok := FindInstance(spec, candidate); !ok {
			return candidate
		}
	}
	return base + "-" + randomDevSecret(4)
}
