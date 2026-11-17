package obelisk

import "fmt"

// KongRouteServiceName is the declarative Kong service name for an instance route.
func KongRouteServiceName(inst ServiceInstance) string {
	return fmt.Sprintf("route-%s-%s", inst.Type, inst.ID)
}

// PublicHTTPPaths returns Kong path prefixes for HTTP-exposed instance types.
// The first path is the canonical path shown in the admin UI.
func PublicHTTPPaths(inst ServiceInstance) []string {
	switch inst.Type {
	case "rustfs":
		paths := []string{"/s3/" + inst.ID}
		if inst.ID == LegacyDefaultInstanceID {
			paths = append(paths, "/s3")
		}
		return paths
	default:
		return nil
	}
}

// PublicHTTPPath returns the canonical public HTTP path prefix for an instance, or "".
func PublicHTTPPath(inst ServiceInstance) string {
	paths := PublicHTTPPaths(inst)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}