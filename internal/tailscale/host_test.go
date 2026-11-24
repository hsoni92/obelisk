package tailscale

import (
	"encoding/json"
	"testing"
)

func TestParseStatusJSON(t *testing.T) {
	raw := `{
	  "BackendState": "Running",
	  "TailscaleIPs": ["100.64.0.1"],
	  "Self": { "DNSName": "box.tailnet.ts.net." },
	  "CurrentTailnet": { "Name": "example.com" }
	}`
	var s statusJSON
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if s.BackendState != "Running" || s.Self.DNSName == "" {
		t.Fatalf("unexpected %+v", s)
	}
}
