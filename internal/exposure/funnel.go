package exposure

import "github.com/hsoni92/obelisk/pkg/obelisk"

// ApplyFunnel enables or disables Tailscale Funnel for the rustfs-s3 Kong route.
// No-op when profile is vps or tailscale is disabled.
func ApplyFunnel(spec obelisk.Spec) error {
	if spec.Deployment.Profile != "local" || !spec.Deployment.Tailscale.Enabled {
		return nil
	}
	if !spec.Exposure.Funnel.RustFSS3.Enabled {
		// TODO: tailscale funnel off
		return nil
	}
	// TODO: tailscale funnel on → Kong proxy /s3
	return nil
}
