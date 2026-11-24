package credentials

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// MergeInstanceCredentials returns inst.Credentials filled from the cluster Secret
// when the on-disk spec omitted secret fields (blueprint redaction, legacy UI saves, etc.).
func MergeInstanceCredentials(ctx context.Context, namespace string, inst obelisk.ServiceInstance) obelisk.InstanceCredentials {
	out := inst.Credentials
	if credentialsComplete(inst.Type, out) {
		return out
	}
	if namespace == "" {
		namespace = "obelisk"
	}
	secret := inst.ReleaseName()
	keys := secretKeysForType(inst.Type)
	if len(keys) == 0 {
		return out
	}
	for logical, dataKey := range keys {
		v := secretData(ctx, namespace, secret, dataKey)
		if v == "" {
			continue
		}
		switch logical {
		case "password":
			if out.Password == "" {
				out.Password = v
			}
		case "secret_key":
			if out.SecretKey == "" {
				out.SecretKey = v
			}
		case "access_key":
			if out.AccessKey == "" {
				out.AccessKey = v
			}
		case "username":
			if out.Username == "" {
				out.Username = v
			}
		}
	}
	return out
}

func credentialsComplete(typ string, c obelisk.InstanceCredentials) bool {
	switch typ {
	case "postgres", "mysql", "mongodb":
		return strings.TrimSpace(c.Password) != ""
	case "redis":
		return strings.TrimSpace(c.Password) != ""
	case "rustfs":
		return strings.TrimSpace(c.AccessKey) != "" && strings.TrimSpace(c.SecretKey) != ""
	case "qdrant":
		return strings.TrimSpace(c.AccessKey) != ""
	case "browserless":
		return strings.TrimSpace(c.AccessKey) != ""
	case "searxng":
		return strings.TrimSpace(c.SecretKey) != ""
	case "redpanda":
		return strings.TrimSpace(c.Username) != "" && strings.TrimSpace(c.Password) != ""
	default:
		return true
	}
}

func secretKeysForType(typ string) map[string]string {
	switch typ {
	case "postgres", "mysql", "mongodb":
		return map[string]string{"password": "password"}
	case "redis":
		return map[string]string{"password": "password"}
	case "rustfs":
		return map[string]string{
			"access_key": "access-key",
			"secret_key": "secret-key",
		}
	case "qdrant":
		return map[string]string{"access_key": "apiKey"}
	case "browserless":
		return map[string]string{"access_key": "token"}
	case "searxng":
		return map[string]string{"secret_key": "secretKey"}
	case "redpanda":
		return map[string]string{
			"username": "username",
			"password": "password",
		}
	default:
		return nil
	}
}

func secretData(ctx context.Context, namespace, name, dataKey string) string {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return ""
	}
	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", name,
		"-n", namespace, "-o", fmt.Sprintf("jsonpath={.data['%s']}", dataKey))
	cmd.Env = kubeEnvForCredentials()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return ""
	}
	dec, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	return strings.TrimSpace(string(dec))
}

func kubeEnvForCredentials() []string {
	// Mirror controller k8s env resolution without importing controller (cycle).
	return appendKubeconfig(os.Environ())
}

func appendKubeconfig(env []string) []string {
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	if strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		return env
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return append(env, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml")
	}
	return env
}
