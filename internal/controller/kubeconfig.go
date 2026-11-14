package controller

import "os"

func kubeconfigEnv() []string {
	kc := os.Getenv("KUBECONFIG")
	if kc == "" {
		kc = "/etc/rancher/k3s/k3s.yaml"
	}
	return append(os.Environ(), "KUBECONFIG="+kc)
}