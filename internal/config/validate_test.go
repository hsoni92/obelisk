package config

import (
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestValidate_VPSNoTailscale(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Deployment.Tailscale.Enabled = true
	if err := Validate(&spec); err == nil {
		t.Fatal("expected error when tailscale on vps")
	}
}

func TestValidate_FunnelRequiresTailscale(t *testing.T) {
	spec := obelisk.DefaultSpec("local")
	spec.Exposure.Funnel.RustFSS3.Enabled = true
	if err := Validate(&spec); err == nil {
		t.Fatal("expected funnel without tailscale to fail")
	}
	spec.Deployment.Tailscale.Enabled = true
	if err := Validate(&spec); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}