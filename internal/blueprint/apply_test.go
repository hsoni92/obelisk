package blueprint

import (
	"path/filepath"
	"testing"

	"github.com/hsoni92/obelisk/internal/config"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestApplyAdoptsBlueprintMachineID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "obelisk.yaml")
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-adopt01"
	spec.Instances = []obelisk.ServiceInstance{{
		ID:      "db",
		Type:    "postgres",
		Enabled: true,
		Credentials: obelisk.InstanceCredentials{
			Username: "obelisk",
			Password: "db-secret",
			Database: "app",
		},
	}}
	if err := config.SaveQuiet(path, spec); err != nil {
		t.Fatal(err)
	}
	rt, err := NewRuntime(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	kdf := testKDF(t)
	key, err := DeriveKey("phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := CollectSecrets(path, spec)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Pack(spec, secrets, key, kdf, 2)
	if err != nil {
		t.Fatal(err)
	}
	unpacked, err := Unpack(bundle.Files, key)
	if err != nil {
		t.Fatal(err)
	}
	if unpacked.Spec.Instances[0].Credentials.Password != "" {
		t.Fatal("public spec still has a password")
	}
	rt.Session.Set(key, kdf)
	local := spec
	local.Machine.ID = "obelisk-local99"
	local.Instances[0].Credentials.Password = ""
	if err := config.SaveQuiet(path, local); err != nil {
		t.Fatal(err)
	}
	ch := &Chain{
		Repository: "octo/obelisk-appliance",
		Branch:     "main",
		Choice:     "adopt",
		Bundle:     unpacked,
		LocalID:    "obelisk-local99",
		RemoteID:   "obelisk-adopt01",
		Mismatch:   true,
	}
	if err := rt.ChainApply(ch); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Machine.ID != "obelisk-adopt01" {
		t.Fatalf("machine id = %s", got.Machine.ID)
	}
	if got.Instances[0].Credentials.Password != "db-secret" {
		t.Fatalf("password = %q", got.Instances[0].Credentials.Password)
	}
	if got.Blueprint.GitHub.Generation != 2 {
		t.Fatalf("generation = %d", got.Blueprint.GitHub.Generation)
	}
}
