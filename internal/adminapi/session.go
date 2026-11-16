package adminapi

import (
	"context"
	"net/http"
)

type githubLoginKey struct{}

func (s *Server) requireGitHub(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = withNormalizedPath(r)
		if s.Blueprint == nil || isPublic(r) {
			next.ServeHTTP(w, r)
			return
		}
		login, ok := s.Blueprint.Authenticate(r)
		if !ok {
			writeJSONStatus(w, http.StatusUnauthorized, map[string]string{"error": "github_login_required"})
			return
		}
		ctx := context.WithValue(r.Context(), githubLoginKey{}, login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublic(r *http.Request) bool {
	switch r.URL.Path {
	case "/api/v1/health":
		return r.Method == http.MethodGet
	case "/api/v1/blueprint/status", "/api/v1/blueprint/github/config":
		return r.Method == http.MethodGet
	case "/api/v1/blueprint/github/authorize", "/api/v1/blueprint/github/callback":
		return r.Method == http.MethodPost
	default:
		return false
	}
}

// loginFromContext is the GitHub login set by requireGitHub.
func loginFromContext(r *http.Request) string {
	v, _ := r.Context().Value(githubLoginKey{}).(string)
	return v
}
