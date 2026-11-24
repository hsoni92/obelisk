package obelisk

import "strings"

const (
	DeploymentRoleLocal   = "local"
	DeploymentRoleGateway = "gateway"
)

// NormalizeDeployment fills defaults for deployment fields.
func NormalizeDeployment(spec *Spec) {
	if spec == nil {
		return
	}
	role := strings.ToLower(strings.TrimSpace(spec.Deployment.Role))
	if role == "" {
		role = DeploymentRoleLocal
	}
	spec.Deployment.Role = role
	if spec.Deployment.Profile == "" {
		spec.Deployment.Profile = "vps"
	}
}

// IsGateway reports whether this appliance is the tailnet connection plane.
func IsGateway(spec Spec) bool {
	return strings.EqualFold(strings.TrimSpace(spec.Deployment.Role), DeploymentRoleGateway)
}

// ConnectionEdgeEnabled is true when per-instance public edge routing (Kong / native) applies.
func ConnectionEdgeEnabled(spec Spec) bool {
	return VPSEdge(spec) || IsGateway(spec)
}
