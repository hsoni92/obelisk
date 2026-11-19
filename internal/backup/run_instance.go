package backup

import (
	"context"
	"strings"
)

// InstanceFilter limits backup to one instance when both fields are set.
type InstanceFilter struct {
	ServiceType string
	InstanceID  string
}

func (f InstanceFilter) active() bool {
	return strings.TrimSpace(f.ServiceType) != "" && strings.TrimSpace(f.InstanceID) != ""
}

// RunForInstance uploads one instance (both type and id required).
func (r *Runner) RunForInstance(ctx context.Context, serviceType, instanceID string) (RunOutcome, error) {
	return r.runScoped(ctx, InstanceFilter{ServiceType: serviceType, InstanceID: instanceID})
}