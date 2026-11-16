package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestK3sKubeconfigNeedsRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode 000 files")
	}
	dir := t.TempDir()
	t.Cleanup(func() { k3sKubeconfigPath = "/etc/rancher/k3s/k3s.yaml" })

	missing := filepath.Join(dir, "missing.yaml")
	k3sKubeconfigPath = missing
	if K3sKubeconfigNeedsRoot() {
		t.Fatal("missing kubeconfig should not require sudo")
	}

	readable := filepath.Join(dir, "readable.yaml")
	if err := os.WriteFile(readable, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	k3sKubeconfigPath = readable
	if K3sKubeconfigNeedsRoot() {
		t.Fatal("readable kubeconfig should not require sudo")
	}

	locked := filepath.Join(dir, "locked.yaml")
	if err := os.WriteFile(locked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	k3sKubeconfigPath = locked
	if !K3sKubeconfigNeedsRoot() {
		t.Fatal("unreadable kubeconfig should require sudo")
	}
}
