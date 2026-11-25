package adminapi

import (
	"net/http/httptest"
	"testing"
)

func TestResolveDiscoveryAudience(t *testing.T) {
	cases := []struct{ url, host, want string }{
		{"/discover/v1/projects/p", "obelisk-kong.obelisk.svc.cluster.local:9080", "in_cluster"},
		{"/discover/v1/projects/p", "obelisk-kong.obelisk.svc:9080", "in_cluster"},
		{"/discover/v1/projects/p", "vps.example.com:9080", "edge"},
		{"/discover/v1/projects/p?audience=edge", "obelisk-kong.obelisk.svc:9080", "edge"},
		{"/discover/v1/projects/p?audience=in_cluster", "vps.example.com", "in_cluster"},
		{"/discover/v1/projects/p?audience=auto", "x.svc.cluster.local", "in_cluster"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", c.url, nil)
		r.Host = c.host
		if got := resolveDiscoveryAudience(r); got != c.want {
			t.Errorf("%s host=%s: got %q want %q", c.url, c.host, got, c.want)
		}
	}
}
