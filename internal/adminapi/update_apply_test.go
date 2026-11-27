package adminapi

import (
	"context"
	"strings"
	"testing"
)

func TestUpdateJobManifest(t *testing.T) {
	name, m := updateJobManifest("obelisk", "img:1", "/opt/obelisk", 7)
	for _, want := range []string{"name: " + name, "namespace: obelisk", "image: img:1", "hostPID: true", "privileged: true", "build-7", "OBELISK_ROOT=/opt/obelisk"} {
		if !strings.Contains(m, want) {
			t.Errorf("manifest missing %q", want)
		}
	}
}

func TestCurrentUpdateApply(t *testing.T) {
	orig := runKubectl
	defer func() { runKubectl = orig }()
	runKubectl = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "logs" {
			return []byte("boom\n"), nil
		}
		return []byte(`{"items":[
			{"metadata":{"name":"old","creationTimestamp":"2026-01-01T00:00:00Z","annotations":{"obelisk.dev/target-build":"3"}},"status":{"succeeded":1}},
			{"metadata":{"name":"new","creationTimestamp":"2026-02-01T00:00:00Z","annotations":{"obelisk.dev/target-build":"7"}},"status":{"failed":1}}]}`), nil
	}
	a := currentUpdateApply(context.Background())
	if a == nil || a.Phase != "failed" || a.TargetBuild != 7 || a.LogTail != "boom" {
		t.Fatalf("got %+v", a)
	}
}
