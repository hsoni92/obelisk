package credentials

import "os"

// k3sKubeconfigPath is the admin kubeconfig k3s writes as root mode 0600.
var k3sKubeconfigPath = "/etc/rancher/k3s/k3s.yaml"

// K3sKubeconfigNeedsRoot reports that this process is on a k3s node and cannot
// read the cluster admin kubeconfig. Callers should re-exec with sudo.
func K3sKubeconfigNeedsRoot() bool {
	if os.Geteuid() == 0 {
		return false
	}
	f, err := os.Open(k3sKubeconfigPath)
	if err == nil {
		_ = f.Close()
		return false
	}
	return !os.IsNotExist(err)
}
