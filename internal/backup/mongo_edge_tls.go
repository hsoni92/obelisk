package backup

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hsoni92/obelisk/internal/controller"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const mongoEdgePEM = "/var/run/edge/tls.pem"

// MongoEdgeTLSEnabled is true when VPS native edge TLS is active in the cluster (mongod requireTLS).
func MongoEdgeTLSEnabled(ctx context.Context, spec obelisk.Spec) bool {
	if !obelisk.VPSEdge(spec) {
		return false
	}
	if controller.ReconcileDisabled() || !kubectlAvailable() {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ns := obeliskNamespace()
	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", "obelisk-edge-tls", "-n", ns)
	cmd.Env = kubeconfigEnv()
	return cmd.Run() == nil
}

// mongoDumpK8sTLSFlags returns mongodump CLI TLS flags for in-pod dumps (legacy --ssl API).
func mongoDumpK8sTLSFlags(edgeTLS bool) string {
	if !edgeTLS {
		return ""
	}
	return fmt.Sprintf(" --ssl --sslCAFile=%s --tlsInsecure", mongoEdgePEM)
}

// mongoRestoreK8sTLSFlags returns mongorestore CLI TLS flags for in-pod restores.
func mongoRestoreK8sTLSFlags(edgeTLS bool) string {
	if !edgeTLS {
		return ""
	}
	return fmt.Sprintf(" --ssl --sslCAFile=%s --tlsInsecure", mongoEdgePEM)
}