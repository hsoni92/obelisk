package blueprint

import (
	"strings"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

func TestScanRejectsSecrets(t *testing.T) {
	if err := ScanPublicYAML([]byte("machine:\n  id: obelisk-abc\n")); err != nil {
		t.Fatal(err)
	}
	if err := ScanPublicYAML([]byte("password: \"\"\n")); err != nil {
		t.Fatal(err)
	}
	if err := ScanPublicYAML([]byte("password: hunter2\n")); err == nil {
		t.Fatal("expected password rejection")
	}
	pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----"
	if err := ScanPublicYAML([]byte(pem)); err == nil {
		t.Fatal("expected PEM rejection")
	}
	if err := ScanPublicYAML([]byte("note: ghp_abcdefghijklmnopqrstuvwxyz0123456789\n")); err == nil {
		t.Fatal("expected token rejection")
	}
}

func TestRedactedSpecRoundTrip(t *testing.T) {
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-round01"
	spec.Instances = []obelisk.ServiceInstance{{
		ID:   "db",
		Type: "postgres",
		Credentials: obelisk.InstanceCredentials{
			Username: "obelisk",
			Password: "s3cret-db",
			Database: "app",
		},
	}}
	pub := spec.RedactedSpec()
	raw, err := yaml.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "s3cret-db") {
		t.Fatalf("public yaml leaked password:\n%s", raw)
	}
	if err := ScanPublicYAML(raw); err != nil {
		t.Fatal(err)
	}
	if spec.Instances[0].Credentials.Password != "s3cret-db" {
		t.Fatal("redaction mutated the live spec")
	}
}
