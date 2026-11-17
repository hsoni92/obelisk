package controller

import (
	"os"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

func writeInstanceValues(inst obelisk.ServiceInstance) (path string, cleanup func(), err error) {
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
	case "searxng":
		creds["secretKey"] = inst.Credentials.SecretKey
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