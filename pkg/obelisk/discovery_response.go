package obelisk

import (
	"strings"
	"time"
)

// ProjectDiscovery is the GET /discover/v1/projects/{id} JSON body.
type ProjectDiscovery struct {
	SchemaVersion int    `json:"schema_version"`
	Project       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
	Profile    string               `json:"profile"`
	ObservedAt string               `json:"observed_at"`
	Notes      string               `json:"notes,omitempty"`
	Instances  []InstanceConnection `json:"instances"`
}

// BuildProjectDiscovery filters BuildConnections output to project membership.
// audience may be ConnectionAudienceInCluster to promote in-cluster URLs to the top level.
func BuildProjectDiscovery(
	spec Spec,
	p Project,
	states map[string]string,
	observedAt time.Time,
	extras ProjectDiscoveryExtras,
	audience string,
) ProjectDiscovery {
	full := BuildConnections(spec, states)
	want := ProjectInstanceSet(p)
	var inst []InstanceConnection
	for _, ic := range full.Instances {
		key := ic.Type + "/" + ic.ID
		if !want[key] {
			continue
		}
		if states != nil {
			if st := states[key]; st != "" {
				_ = st // reserved for future state field on InstanceConnection
			}
		}
		inst = append(inst, InstanceConnectionForAudience(ic, audience))
	}
	for _, ref := range p.InstanceIDs {
		if !want[ref] || !IsExposureMemberRef(ref) {
			continue
		}
		if ic, ok := buildExposureConnection(ref, extras); ok {
			inst = append(inst, InstanceConnectionForAudience(ic, audience))
		}
	}
	var out ProjectDiscovery
	out.SchemaVersion = 1
	out.Project.ID = p.ID
	out.Project.Name = p.Name
	out.Profile = spec.Deployment.Profile
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	out.ObservedAt = observedAt.UTC().Format(time.RFC3339)
	out.Notes = full.Notes
	if strings.EqualFold(strings.TrimSpace(audience), ConnectionAudienceInCluster) && full.InClusterNotes != "" {
		out.Notes = full.InClusterNotes
	}
	if len(inst) == 0 && len(p.InstanceIDs) > 0 {
		if out.Notes != "" {
			out.Notes += " "
		}
		out.Notes += "No enabled instances in this project; check Services."
	}
	out.Instances = inst
	return out
}