package discovery

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const tokenDataKey = "token"

// SecretName is the Kubernetes Secret for a project discovery token.
func SecretName(projectID string) string {
	return "obelisk-discovery-" + projectID
}

// ReadToken loads the bearer token from the cluster, or "" if missing.
func ReadToken(ctx context.Context, namespace, projectID string) string {
	if namespace == "" {
		namespace = "obelisk"
	}
	return secretData(ctx, namespace, SecretName(projectID), tokenDataKey)
}

// WriteToken upserts the discovery token secret.
func WriteToken(ctx context.Context, namespace, projectID, token string) error {
	if namespace == "" {
		namespace = "obelisk"
	}
	name := SecretName(projectID)
	args := []string{
		"create", "secret", "generic", name,
		"-n", namespace,
		"--from-literal=" + tokenDataKey + "=" + token,
		"--dry-run=client", "-o", "yaml",
	}
	create := exec.CommandContext(ctx, "kubectl", args...)
	create.Env = kubeEnv()
	apply := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	apply.Env = kubeEnv()
	pipe, err := create.StdoutPipe()
	if err != nil {
		return err
	}
	apply.Stdin = pipe
	create.Stderr = os.Stderr
	apply.Stderr = os.Stderr
	if err := create.Start(); err != nil {
		return err
	}
	if err := apply.Start(); err != nil {
		_ = create.Process.Kill()
		_ = create.Wait()
		return err
	}
	if err := create.Wait(); err != nil {
		return fmt.Errorf("kubectl create secret: %w", err)
	}
	if err := apply.Wait(); err != nil {
		return fmt.Errorf("kubectl apply secret: %w", err)
	}
	return nil
}

// DeleteSecret removes the discovery token secret (best-effort).
func DeleteSecret(ctx context.Context, namespace, projectID string) {
	if namespace == "" {
		namespace = "obelisk"
	}
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "secret", SecretName(projectID),
		"-n", namespace, "--ignore-not-found=true")
	cmd.Env = kubeEnv()
	_ = cmd.Run()
}

// ValidToken compares bearer tokens in constant time.
func ValidToken(stored, provided string) bool {
	stored = strings.TrimSpace(stored)
	provided = strings.TrimSpace(provided)
	if stored == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) == 1
}

func secretData(ctx context.Context, namespace, name, dataKey string) string {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return ""
	}
	cmd := exec.CommandContext(ctx, "kubectl", "get", "secret", name,
		"-n", namespace, "-o", fmt.Sprintf("jsonpath={.data['%s']}", dataKey))
	cmd.Env = kubeEnv()
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

func kubeEnv() []string {
	env := os.Environ()
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