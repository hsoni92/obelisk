package adminapi

import (
	"net/http"
	"strings"
)

// NormalizeAPIPath repairs Kong strip_path and older admin UI base paths.
func NormalizeAPIPath(p string) string {
	switch {
	case strings.HasPrefix(p, "/api/v1/") || p == "/api/v1":
		return p
	case strings.HasPrefix(p, "/api/"):
		return "/api/v1" + strings.TrimPrefix(p, "/api")
	case strings.HasPrefix(p, "/v1/") || p == "/v1":
		// Kong strip_path on route /api sends /v1/... upstream.
		return "/api" + p
	default:
		return p
	}
}

func withNormalizedPath(r *http.Request) *http.Request {
	np := NormalizeAPIPath(r.URL.Path)
	if np == r.URL.Path {
		return r
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = np
	return r2
}

// normalizeLegacyAPIPath repairs Kong strip_path and older admin UI base paths.
func normalizeLegacyAPIPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, withNormalizedPath(r))
	})
}
