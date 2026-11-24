package backup

import (
	"strings"
	"testing"
)

func TestMongoDumpK8sTLSFlags(t *testing.T) {
	if got := mongoDumpK8sTLSFlags(false); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got := mongoDumpK8sTLSFlags(true)
	for _, want := range []string{"--ssl", "--sslCAFile=" + mongoEdgePEM, "--tlsInsecure"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "--tlsCAFile") {
		t.Fatalf("mongodump uses --sslCAFile not --tlsCAFile: %q", got)
	}
}
