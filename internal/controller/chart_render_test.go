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

func TestPostgresChartRender_NoSslNegotiation(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(root, "..", ".."))
	tpl := filepath.Join(repoRoot, "deploy/helm/services/postgres/templates/statefulset.yaml")
	data, err := os.ReadFile(tpl)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sslnegotiation") {
		t.Fatal("postgres chart must not set sslnegotiation on server")
	}
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not installed")
	}
	chart := filepath.Join(repoRoot, "deploy/helm/services/postgres")
	inst := obelisk.ServiceInstance{
		Type:    "postgres",
		ID:      "test",
		Enabled: true,
		Credentials: obelisk.InstanceCredentials{
			Username: "obelisk",
			Password: "secret",
			Database: "default",
		},
	}
	valuesPath, cleanup, err := writeInstanceValues(true, true, inst)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	r := Reconciler{HelmRoot: repoRoot, Namespace: "obelisk"}
	if err := r.helmTemplatePreflight(context.Background(), chart, "obelisk-postgres-test", "obelisk", valuesPath); err != nil {
		t.Fatal(err)
	}
}
