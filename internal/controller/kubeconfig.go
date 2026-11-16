package controller

import (
	"os"
	"strings"
)

// KubeconfigEnv returns process environment for helm/kubectl subprocesses.
func KubeconfigEnv() []string {
	return kubeconfigEnv()
}

func kubeconfigEnv() []string {
	env := os.Environ()
	if strings.TrimSpace(os.Getenv("KUBECONFIG")) != "" {
		return env
	}
	// In-cluster: use mounted service-account credentials (no KUBECONFIG file).
	if strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		return env
	}
	if _, err := os.Stat("/etc/rancher/k3s/k3s.yaml"); err == nil {
		return append(env, "KUBECONFIG=/etc/rancher/k3s/k3s.yaml")
	}
	return env
}