package diagnostics

import (
	"context"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestRun_skipsDisabled(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{{
		ID: "x", Type: "redis", Enabled: false,
	}}
	resp := Run(context.Background(), spec, obelisk.Connections{}, nil, []InstanceRef{{Type: "redis", ID: "x"}})
	if len(resp.Results) != 1 {
		t.Fatalf("len=%d", len(resp.Results))
	}
	if !resp.Results[0].Skipped || resp.Results[0].OK {
		t.Fatalf("%+v", resp.Results[0])
	}
}