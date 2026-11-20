package backup

import (
	"context"
	"fmt"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// RestoreInstanceData applies a backup artifact to a running instance (same format as CollectDataDumps).
func RestoreInstanceData(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	if !obelisk.ServiceTypeSupportsBackup(inst.Type) {
		return fmt.Errorf("restore not supported for %q", inst.Type)
	}
	if len(dump) == 0 {
		return fmt.Errorf("backup payload is empty")
	}
	if K8sDataBackupEnabled() {
		spec, err := config.Load(config.EnsureReadablePath(""))
		if err != nil {
			return err
		}
		return restoreInstanceK8s(ctx, spec, inst, dump)
	}
	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		return restoreInstanceLocal(ctx, inst, dump)
	default:
		return fmt.Errorf(
			"restore for %q requires a running Kubernetes cluster (run make dev)",
			inst.Type,
		)
	}
}

// RestorePostgresSQL applies a plain SQL dump to a running postgres instance.
func RestorePostgresSQL(ctx context.Context, inst obelisk.ServiceInstance, dump []byte) error {
	return RestoreInstanceData(ctx, inst, dump)
}