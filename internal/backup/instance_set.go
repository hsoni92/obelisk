package backup

import (
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// InstanceSet limits a backup run to specific instances (keys "type/id"). Empty means all.
type InstanceSet map[string]struct{}

// InstanceKey is the canonical map key for type/id.
func InstanceKey(serviceType, instanceID string) string {
	return strings.ToLower(strings.TrimSpace(serviceType)) + "/" + strings.TrimSpace(instanceID)
}

// Add registers one instance in the set.
func (s InstanceSet) Add(serviceType, instanceID string) {
	if s == nil {
		return
	}
	s[InstanceKey(serviceType, instanceID)] = struct{}{}
}

// Contains reports whether type/id is in the set; empty set matches everything.
func (s InstanceSet) Contains(serviceType, instanceID string) bool {
	if len(s) == 0 {
		return true
	}
	_, ok := s[InstanceKey(serviceType, instanceID)]
	return ok
}

func filterDumpsBySet(dumps []DataDump, only InstanceSet) []DataDump {
	if len(only) == 0 {
		return dumps
	}
	out := make([]DataDump, 0, len(dumps))
	for _, d := range dumps {
		if only.Contains(d.ServiceType, d.InstanceID) {
			out = append(out, d)
		}
	}
	return out
}

func countEligibleDataInstancesInSet(spec obelisk.Spec, states map[string]string, only InstanceSet) int {
	if len(only) == 0 {
		return countEligibleDataInstances(spec, states)
	}
	n := 0
	for _, inst := range spec.Instances {
		if !only.Contains(inst.Type, inst.ID) {
			continue
		}
		if !inst.Enabled || !obelisk.InstanceDataBackupEnabled(inst) {
			continue
		}
		if states != nil {
			st := states[inst.StateKey()]
			if st != "" && st != "running" {
				continue
			}
		}
		n++
	}
	return n
}
