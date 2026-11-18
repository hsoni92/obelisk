package exposure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

const nativeEdgeConfigMap = "native-edge-config"
const nativeEdgeDeployment = "obelisk-native-edge"

// RenderNativeEdgeNginx emits an nginx stream config that routes on TLS SNI (L4 passthrough).
func RenderNativeEdgeNginx(spec obelisk.Spec, namespace string) string {
	if namespace == "" {
		namespace = "obelisk"
	}
	if !obelisk.VPSEdge(spec) {
		return defaultNativeEdgePlaceholder()
	}
	var b strings.Builder
	b.WriteString("worker_processes auto;\n")
	b.WriteString("error_log /var/log/nginx/error.log warn;\n")
	b.WriteString("events { worker_connections 2048; }\n")
	b.WriteString("stream {\n")
	b.WriteString("    resolver kube-dns.kube-system.svc.cluster.local valid=10s ipv6=off;\n")
	b.WriteString("    map $ssl_preread_server_name $native_backend {\n")
	b.WriteString("        default \"\";\n")
	for _, inst := range spec.Instances {
		if !inst.Enabled {
			continue
		}
		port, ok := obelisk.NativeEdgeTypes[inst.Type]
		if !ok {
			continue
		}
		backend := fmt.Sprintf("%s.%s.svc.cluster.local:%d", inst.ReleaseName(), namespace, port)
		for _, host := range obelisk.KongEdgeHosts(inst) {
			fmt.Fprintf(&b, "        %s %s;\n", host, backend)
		}
	}
	b.WriteString("    }\n")
	b.WriteString("    server {\n")
	b.WriteString("        listen 8443;\n")
	b.WriteString("        listen [::]:8443;\n")
	b.WriteString("        ssl_preread on;\n")
	b.WriteString("        proxy_pass $native_backend;\n")
	b.WriteString("        proxy_connect_timeout 15s;\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")
	return b.String()
}

func defaultNativeEdgePlaceholder() string {
	return strings.TrimSpace(`
worker_processes auto;
events { worker_connections 256; }
stream {
    server { listen 8443; return 444; }
}
`) + "\n"
}

// ReconcileNativeEdge updates the nginx stream proxy for native TLS data services.
func ReconcileNativeEdge(ctx context.Context, spec obelisk.Spec, namespace string) error {
	if namespace == "" {
		namespace = "obelisk"
	}
	if !obelisk.VPSEdge(spec) || !NeedsNativeEdgeTLS(spec) {
		return nil
	}
	body := []byte(RenderNativeEdgeNginx(spec, namespace))
	current, err := nativeEdgeConfigFromCluster(ctx, namespace)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current) == strings.TrimSpace(string(body)) {
		return nil
	}
	if err := applyNativeEdgeConfig(ctx, namespace, body); err != nil {
		return err
	}
	return restartNativeEdge(ctx, namespace)
}

func nativeEdgeConfigFromCluster(ctx context.Context, namespace string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", "get", "configmap", nativeEdgeConfigMap,
		"-n", namespace, "-o", "jsonpath={.data.nginx\\.conf}")
	cmd.Env = kubeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found") {
			return "", nil
		}
		return "", fmt.Errorf("get native edge configmap: %w: %s", err, msg)
	}
	return string(out), nil
}

func applyNativeEdgeConfig(ctx context.Context, namespace string, body []byte) error {
	tmp, err := os.CreateTemp("", "obelisk-native-edge-*.conf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	create := exec.CommandContext(ctx, "kubectl", "create", "configmap", nativeEdgeConfigMap,
		"-n", namespace, "--from-file=nginx.conf="+tmp.Name(),
		"--dry-run=client", "-o", "yaml")
	create.Env = kubeEnv()
	manifest, err := create.Output()
	if err != nil {
		if out, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("render native edge configmap: %w: %s", err, strings.TrimSpace(string(out.Stderr)))
		}
		return err
	}

	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	apply.Env = kubeEnv()
	apply.Stdin = bytes.NewReader(manifest)
	out, err := apply.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply native edge configmap: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func restartNativeEdge(ctx context.Context, namespace string) error {
	ok, err := deploymentExists(ctx, namespace, nativeEdgeDeployment)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	cmd := exec.CommandContext(ctx, "kubectl", "rollout", "restart",
		"deployment/"+nativeEdgeDeployment, "-n", namespace)
	cmd.Env = kubeEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart native edge: %w: %s", err, strings.TrimSpace(string(out)))
	}
	wait := exec.CommandContext(ctx, "kubectl", "rollout", "status",
		"deployment/"+nativeEdgeDeployment, "-n", namespace, "--timeout=180s")
	wait.Env = kubeEnv()
	if wout, werr := wait.CombinedOutput(); werr != nil {
		return fmt.Errorf("native edge rollout: %w: %s", werr, strings.TrimSpace(string(wout)))
	}
	return nil
}