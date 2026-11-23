package controller

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestSearxngChartRender_JSONConfigSeed(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(root, "..", ".."))
	tpl := filepath.Join(repoRoot, "deploy/helm/services/searxng/templates/statefulset.yaml")
	data, err := os.ReadFile(tpl)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "seed-config") {
		t.Fatal("searxng chart must seed settings with JSON format support")
	}
	if !strings.Contains(body, "format=json") && !strings.Contains(body, "- json") {
		t.Fatal("searxng seed script must enable json output format")
	}
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not installed")
	}
	chart := filepath.Join(repoRoot, "deploy/helm/services/searxng")
	inst := obelisk.ServiceInstance{
		Type:    "searxng",
		ID:      "test",
		Enabled: true,
		Credentials: obelisk.InstanceCredentials{
			SecretKey: "secret",
		},
	}
	valuesPath, cleanup, err := writeInstanceValues(true, true, inst)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := os.ReadFile(valuesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "baseUrl:") {
		t.Fatalf("searxng helm values must set baseUrl:\n%s", raw)
	}

	r := Reconciler{HelmRoot: repoRoot, Namespace: "obelisk"}
	if err := r.helmTemplatePreflight(context.Background(), chart, "obelisk-searxng-test", "obelisk", valuesPath); err != nil {
		t.Fatal(err)
	}
}