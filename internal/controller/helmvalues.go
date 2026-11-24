package controller

import (
	"fmt"
	"os"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

func writeInstanceValues(vpsEdge, edgeTLSReady bool, inst obelisk.ServiceInstance) (path string, cleanup func(), err error) {
	creds := map[string]string{}
	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		creds["username"] = inst.Credentials.Username
		creds["password"] = inst.Credentials.Password
		creds["database"] = inst.Credentials.Database
	case "redis":
		creds["password"] = inst.Credentials.Password
	case "rustfs":
		creds["accessKey"] = inst.Credentials.AccessKey
		creds["secretKey"] = inst.Credentials.SecretKey
	case "qdrant":
		creds["apiKey"] = inst.Credentials.AccessKey
	case "browserless":
		creds["token"] = inst.Credentials.AccessKey
	case "searxng":
		creds["secretKey"] = inst.Credentials.SecretKey
	case "redpanda":
		creds["username"] = inst.Credentials.Username
		creds["password"] = inst.Credentials.Password
	}
	replicas := 1
	if !inst.Enabled {
		replicas = 0
	}
	vals := map[string]any{
		"credentials": creds,
		"replicas":    replicas,
	}
	if inst.Type == "rustfs" {
		if d := obelisk.RustFSServerDomains(inst); d != "" {
			vals["serverDomains"] = d
		}
	}
	if inst.Type == "redpanda" && inst.Enabled && strings.TrimSpace(os.Getenv("OBELISK_DEV")) != "" {
		if h := obelisk.PublicEdgeHost(inst); h != "" {
			vals["devAdvertiseHost"] = h
		}
	}
	if inst.Type == "searxng" && inst.Enabled {
		if vpsEdge {
			if u := obelisk.PublicHTTPEdgeURL(inst); u != "" {
				vals["baseUrl"] = strings.TrimRight(u, "/") + "/"
			}
		} else {
			vals["baseUrl"] = fmt.Sprintf("http://%s:8080/", inst.ReleaseName())
		}
	}
	if inst.Type == "unimeter" && inst.Enabled && strings.TrimSpace(os.Getenv("OBELISK_DEV")) != "" {
		// Partition map must include host:port; long *.localhost MY_ADDR values are truncated by v0.1 server builds.
		vals["myAddr"] = "127.0.0.1:7001"
	}
	if vpsEdge && edgeTLSReady && nativeEdgeTLSInCluster(inst.Type) {
		vals["edgeTLS"] = map[string]any{
			"enabled":    true,
			"secretName": "obelisk-edge-tls",
		}
	}
	data, err := yaml.Marshal(vals)
	if err != nil {
		return "", func() {}, err
	}
	f, err := os.CreateTemp("", "obelisk-helm-"+inst.ID+"-*.yaml")
	if err != nil {
		return "", func() {}, err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", func() {}, err
	}
	path = f.Name()
	_ = f.Close()
	return path, func() { _ = os.Remove(path) }, nil
}

func nativeEdgeTLSInCluster(typ string) bool {
	switch typ {
	case "postgres", "mysql", "mongodb", "redis":
		return true
	default:
		return false
	}
}
