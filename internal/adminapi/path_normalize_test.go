package adminapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeLegacyAPIPath_KongStrip(t *testing.T) {
	var got string
	h := normalizeLegacyAPIPath(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got != "/api/v1/health" {
		t.Fatalf("path=%q want /api/v1/health", got)
	}
}