package controller

import (
	"strings"
	"testing"
)

func TestMongoMongoshTLSFlags(t *testing.T) {
	if got := mongoMongoshTLSFlags(false); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got := mongoMongoshTLSFlags(true)
	for _, want := range []string{"--tls ", mongoEdgePEM, "--tlsCAFile=" + mongoEdgePEM, "--tlsAllowInvalidCertificates"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}