package obelisk

import "strings"

// SpecWithResolvedCredentials returns a copy of spec with per-instance credentials
// filled when resolver supplies non-empty fields (typically from cluster Secrets).
func SpecWithResolvedCredentials(spec Spec, resolver func(ServiceInstance) InstanceCredentials) Spec {
	out := spec
	if resolver == nil || len(spec.Instances) == 0 {
		return out
	}
	out.Instances = append([]ServiceInstance(nil), spec.Instances...)
	for i := range out.Instances {
		got := resolver(out.Instances[i])
		mergeCredentials(&out.Instances[i].Credentials, got)
	}
	return out
}

func mergeCredentials(dst *InstanceCredentials, src InstanceCredentials) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(dst.Username) == "" {
		dst.Username = src.Username
	}
	if strings.TrimSpace(dst.Password) == "" {
		dst.Password = src.Password
	}
	if strings.TrimSpace(dst.Database) == "" {
		dst.Database = src.Database
	}
	if strings.TrimSpace(dst.AccessKey) == "" {
		dst.AccessKey = src.AccessKey
	}
	if strings.TrimSpace(dst.SecretKey) == "" {
		dst.SecretKey = src.SecretKey
	}
}