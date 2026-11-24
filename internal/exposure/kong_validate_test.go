package exposure

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// TestKongMergedParsesWithKongImage requires docker and Kong 3.4 (set KONG_VALIDATE=1).
func TestKongMergedParsesWithKongImage(t *testing.T) {
	if os.Getenv("KONG_VALIDATE") == "" {
		t.Skip("set KONG_VALIDATE=1 to run docker kong config parse")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}
	t.Setenv("OBELISK_PUBLIC_HOST", "178.105.35.230")
	spec := obelisk.DefaultSpec("vps")
	spec.Instances = []obelisk.ServiceInstance{
		{ID: "default", Type: "redis", Enabled: true},
		{ID: "default", Type: "rustfs", Enabled: true},
	}
	base, err := os.ReadFile(filepath.Join("..", "..", "deploy", "helm", "platform", "files", "kong-dbless.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := mergeInstanceRoutes(base, spec, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "kong.yaml")
	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("docker", "run", "--rm",
		"-e", "KONG_DATABASE=off",
		"-v", p+":/kong/kong.yaml:ro",
		"kong:3.4", "kong", "config", "parse", "/kong/kong.yaml")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kong parse: %v\n%s", err, b)
	}
	combined := string(b)
	if !strings.Contains(combined, "parse successful") {
		t.Fatalf("unexpected kong output:\n%s", combined)
	}
}
