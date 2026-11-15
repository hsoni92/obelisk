package backup

import (
	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Runner orchestrates control-plane and data-plane backups to S3-compatible storage.
type Runner struct {
	ConfigPath     string
	InstanceStates map[string]string
}

func (r *Runner) loadSpec() (obelisk.Spec, string, error) {
	cfg := config.EnsureReadablePath(r.ConfigPath)
	spec, err := config.Load(cfg)
	if err != nil {
		return obelisk.Spec{}, cfg, err
	}
	return spec, cfg, nil
}

// Ready reports whether remote target metadata and credentials are configured.
func (r *Runner) Ready() (bool, string, error) {
	spec, cfg, err := r.loadSpec()
	if err != nil {
		return false, "", err
	}
	if !spec.BackupRemoteReady() {
		return false, "add at least one enabled backup destination in DR", nil
	}
	ok, err := AnyCredentialsConfigured(cfg)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "configure R2/S3 access keys for a destination in DR", nil
	}
	return true, "", nil
}

// CredentialsConfigured is true when a credential file exists beside the config.
func CredentialsConfigured(configPath string) (bool, error) {
	_, ok, err := LoadCredentials(configPath)
	return ok, err
}
