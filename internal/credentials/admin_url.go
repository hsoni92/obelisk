package credentials

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// DefaultAdminPublicURL guesses the admin UI origin operators use in a browser.
// Order: OBELISK_PUBLIC_URL, OBELISK_KONG_URL, Kong Service via kubectl, first host IP.
func DefaultAdminPublicURL(ctx context.Context) string {
	if u := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	if u := strings.TrimSpace(os.Getenv("OBELISK_KONG_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	if base := kongURLFromKubectl(ctx); base != "" {
		return strings.TrimRight(base, "/")
	}
	if ip := firstHostIP(); ip != "" {
		return fmt.Sprintf("http://%s:%s", ip, obelisk.AdminHTTPPortString())
	}
	return ""
}

func kongURLFromKubectl(ctx context.Context) string {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return ""
	}
	kc := strings.TrimSpace(os.Getenv("KUBECONFIG"))
	if kc == "" {
		kc = "/etc/rancher/k3s/k3s.yaml"
	}
	if _, err := os.Stat(kc); err != nil {
		return ""
	}
	ns := envOrDefault("OBELISK_NAMESPACE", "obelisk")
	port := kubectlJSON(ctx, kc, ns, `jsonpath={.spec.ports[0].port}`)
	if port == "" {
		port = obelisk.AdminHTTPPortString()
	}
	if lb := kubectlJSON(ctx, kc, ns, `jsonpath={.status.loadBalancer.ingress[0].ip}`); lb != "" {
		return fmt.Sprintf("http://%s:%s", lb, port)
	}
	if ext := kubectlJSON(ctx, kc, ns, `jsonpath={.spec.externalIPs[0]}`); ext != "" {
		return fmt.Sprintf("http://%s:%s", ext, port)
	}
	if ip := firstHostIP(); ip != "" {
		return fmt.Sprintf("http://%s:%s", ip, port)
	}
	return ""
}

func kubectlJSON(ctx context.Context, kubeconfig, ns, jsonpath string) string {
	out, err := exec.CommandContext(ctx, "kubectl",
		"--kubeconfig", kubeconfig,
		"get", "svc", "-n", ns, "obelisk-kong",
		"-o", jsonpath,
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func firstHostIP() string {
	out, err := exec.Command("hostname", "-I").Output()
	if err != nil {
		return ""
	}
	for _, ip := range strings.Fields(string(out)) {
		if skipHostIP(ip) {
			continue
		}
		return ip
	}
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func skipHostIP(ip string) bool {
	return strings.HasPrefix(ip, "10.42.") ||
		strings.HasPrefix(ip, "10.43.") ||
		strings.HasPrefix(ip, "127.") ||
		strings.HasPrefix(ip, "169.254.")
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}