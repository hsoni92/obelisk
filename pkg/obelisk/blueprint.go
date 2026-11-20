package obelisk

import (
	"fmt"
	"strings"
)

// Sync status values stored on blueprint.github.sync_status.
const (
	SyncIdle       = "idle"
	SyncPending    = "pending"
	SyncSynced     = "synced"
	SyncFailed     = "failed"
	SyncLocked     = "locked"
	SyncPushPaused = "push_paused"
)

// DefaultBlueprintRepoName is created when onboarding does not name a repo.
const DefaultBlueprintRepoName = "obelisk-appliance"

// BlueprintBranchPrefix is prepended to machine.id for the default git upstream ref.
const BlueprintBranchPrefix = "machine-"

// Blueprint is the encrypted GitHub mirror of desired state.
type Blueprint struct {
	GitHub BlueprintGitHub `yaml:"github,omitempty" json:"github,omitempty"`
}

// BlueprintGitHub is non-secret mirror metadata. OAuth tokens are not stored here.
type BlueprintGitHub struct {
	Repository     string `yaml:"repository,omitempty" json:"repository,omitempty"`
	Branch         string `yaml:"branch,omitempty" json:"branch,omitempty"`
	Generation     int    `yaml:"generation,omitempty" json:"generation,omitempty"`
	LastSyncAt     string `yaml:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	LastCommitSHA  string `yaml:"last_commit_sha,omitempty" json:"last_commit_sha,omitempty"`
	SyncStatus     string `yaml:"sync_status,omitempty" json:"sync_status,omitempty"`
	CryptoUnlocked bool   `yaml:"crypto_unlocked,omitempty" json:"crypto_unlocked,omitempty"`
	OAuthLogin     string `yaml:"oauth_login,omitempty" json:"oauth_login,omitempty"`
	// KDF parameters are not secret. They let unlock re-derive the key after reboot
	// without reading GitHub first. The passphrase itself is never written here.
	KDFSalt        string `yaml:"kdf_salt,omitempty" json:"kdf_salt,omitempty"`
	KDFMemory      uint32 `yaml:"kdf_memory,omitempty" json:"kdf_memory,omitempty"`
	KDFIterations  uint32 `yaml:"kdf_iterations,omitempty" json:"kdf_iterations,omitempty"`
	KDFParallelism uint8  `yaml:"kdf_parallelism,omitempty" json:"kdf_parallelism,omitempty"`
	KeyID          string `yaml:"key_id,omitempty" json:"key_id,omitempty"`
	LastPushError  string `yaml:"last_push_error,omitempty" json:"last_push_error,omitempty"`
}

// BlueprintUpstreamBranch returns the default git ref for an appliance upstream.
func BlueprintUpstreamBranch(machineID string) (string, error) {
	machineID = strings.TrimSpace(machineID)
	if machineID == "" {
		return "", fmt.Errorf("machine id required")
	}
	if err := ValidateMachineID(machineID); err != nil {
		return "", err
	}
	return BlueprintBranchPrefix + machineID, nil
}

// EffectiveBlueprintBranch returns the branch used for push and remote probes.
// Legacy installs may still have branch "main"; per-machine upstream is machine-<machine.id>.
func EffectiveBlueprintBranch(spec Spec) string {
	branch := strings.TrimSpace(spec.Blueprint.GitHub.Branch)
	repo := strings.TrimSpace(spec.Blueprint.GitHub.Repository)
	if repo != "" {
		if upstream, err := BlueprintUpstreamBranch(spec.Machine.ID); err == nil {
			if branch == "" || branch == "main" {
				return upstream
			}
		}
	}
	if branch != "" {
		return branch
	}
	if upstream, err := BlueprintUpstreamBranch(spec.Machine.ID); err == nil {
		return upstream
	}
	return "main"
}

// NormalizeBlueprint fills defaults and clears the on-disk unlock hint.
// Real unlock state lives only in controller memory.
func NormalizeBlueprint(spec *Spec) {
	spec.Blueprint.GitHub.CryptoUnlocked = false
	spec.Blueprint.GitHub.Repository = strings.TrimSpace(spec.Blueprint.GitHub.Repository)
	spec.Blueprint.GitHub.Branch = strings.TrimSpace(spec.Blueprint.GitHub.Branch)
	spec.Blueprint.GitHub.OAuthLogin = strings.TrimSpace(spec.Blueprint.GitHub.OAuthLogin)
	if spec.Blueprint.GitHub.Repository == "" {
		return
	}
	if spec.Blueprint.GitHub.Branch == "" || spec.Blueprint.GitHub.Branch == "main" {
		if upstream, err := BlueprintUpstreamBranch(spec.Machine.ID); err == nil {
			spec.Blueprint.GitHub.Branch = upstream
		} else if spec.Blueprint.GitHub.Branch == "" {
			spec.Blueprint.GitHub.Branch = "main"
		}
	}
	switch spec.Blueprint.GitHub.SyncStatus {
	case SyncIdle, SyncPending, SyncSynced, SyncFailed, SyncLocked, SyncPushPaused:
	default:
		spec.Blueprint.GitHub.SyncStatus = SyncIdle
	}
}

// RedactedSpec returns a copy safe to commit as obelisk.public.yaml.
// Secret credential fields are cleared. Sidecar files are not part of the spec.
func (s Spec) RedactedSpec() Spec {
	out := s
	out.Blueprint.GitHub.CryptoUnlocked = false
	out.Blueprint.GitHub.LastPushError = ""
	if len(s.Instances) == 0 {
		return out
	}
	out.Instances = append([]ServiceInstance(nil), s.Instances...)
	for i := range out.Instances {
		out.Instances[i].Credentials.Password = ""
		out.Instances[i].Credentials.SecretKey = ""
		out.Instances[i].Credentials.AccessKey = ""
	}
	return out
}