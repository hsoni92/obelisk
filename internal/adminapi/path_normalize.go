package adminapi

import (
	"net/http"
	"strings"
)

// normalizeLegacyAPIPath rewrites /api/foo → /api/v1/foo for older admin UI bundles.
func normalizeLegacyAPIPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/api/v1/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/api/v1" + strings.TrimPrefix(p, "/api")
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}