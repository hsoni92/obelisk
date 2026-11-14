package backup

import (
	"fmt"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Runner orchestrates control-plane and data-plane backups to S3-compatible storage.
type Runner struct {
	ConfigPath string
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
		return false, "configure backup.remote endpoint and buckets in DR", nil
	}
	ok, err := CredentialsConfigured(cfg)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "configure R2/S3 access keys in DR", nil
	}
	return true, "", nil
}

// CredentialsConfigured is true when a credential file exists beside the config.
func CredentialsConfigured(configPath string) (bool, error) {
	_, ok, err := LoadCredentials(configPath)
	return ok, err
}

func (r *Runner) RunNow() error {
	ready, msg, err := r.Ready()
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("backup not configured: %s", msg)
	}
	// v1: CronJob/worker uploads; manual run is accepted once preflight passes.
	return nil
}