package openroutergateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

var allowedPathPrefixes = []string{
	"/api/v1/chat/completions",
	"/api/v1/completions",
	"/api/v1/embeddings",
	"/api/v1/models",
	"/api/v1/generation",
}

func pathAllowed(path string) bool {
	for _, p := range allowedPathPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func bodyStreams(body []byte) bool {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	v, ok := m["stream"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func retryUpstreamStatus(code int) bool {
	switch code {
	case http.StatusPaymentRequired, http.StatusTooManyRequests:
		return true
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return false
	default:
		return false
	}
}

func copyHeaders(dst, src http.Header, upstreamKey string) {
	for k, vals := range src {
		kl := strings.ToLower(k)
		if kl == "authorization" || kl == "host" || kl == "connection" {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
	dst.Set("Authorization", "Bearer "+upstreamKey)
}

func (s *Server) doUpstream(r *http.Request, body []byte, acc upstreamAccount, urlPath string) (*http.Response, error) {
	upstreamURL := openRouterBase + urlPath
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyHeaders(req.Header, r.Header, acc.Key)
	return s.client.Do(req)
}

func writeGatewayJSON(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func forwardResponse(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
