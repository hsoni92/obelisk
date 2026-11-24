package backup

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestK8sRustfsQdrantDumpsIntegration(t *testing.T) {
	if os.Getenv("OBELISK_BACKUP_INTEGRATION") != "1" {
		t.Skip("set OBELISK_BACKUP_INTEGRATION=1 to run against local cluster")
	}
	if !kubectlAvailable() {
		t.Skip("kubectl not available")
	}
	ctx := context.Background()
	ns := obeliskNamespace()

	rustBody, err := rustfsDumpK8s(ctx, ns, "obelisk-rustfs-pqr-0")
	if err != nil {
		t.Fatalf("rustfs: %v", err)
	}
	if len(rustBody) < 512 {
		t.Fatalf("rustfs tar too small: %d", len(rustBody))
	}

	key := qdrantKeyFromCluster(t, ns, "obelisk-qdrant-pqr-2-0")
	qBody, err := qdrantDumpK8s(ctx, ns, "obelisk-qdrant-pqr-2-0", key)
	if err != nil {
		t.Fatalf("qdrant: %v", err)
	}
	if len(qBody) < 100 {
		t.Fatalf("qdrant snapshot too small: %d", len(qBody))
	}
}

func qdrantKeyFromCluster(t *testing.T, ns, pod string) string {
	t.Helper()
	out, err := exec.Command("kubectl", "exec", "-n", ns, pod, "-c", "qdrant", "--",
		"printenv", "QDRANT__SERVICE__API_KEY").Output()
	if err != nil {
		t.Fatalf("read qdrant api key: %v", err)
	}
	return string(out[:len(out)-1]) // trim newline
}
