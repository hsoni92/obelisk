package backup

import (
	"encoding/json"
	"testing"
)

func TestParseQdrantCreateSnapshotResp(t *testing.T) {
	raw := `{"result":{"name":"full-snapshot-2026-11-16.snapshot","size":2048},"status":"ok"}`
	var parsed qdrantCreateSnapshotResp
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Result.Name != "full-snapshot-2026-11-16.snapshot" {
		t.Fatalf("name=%q", parsed.Result.Name)
	}
}
