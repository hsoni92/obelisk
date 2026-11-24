package blueprint

import (
	"errors"
	"testing"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

func TestUnpackWrongPassphraseVsCorrupt(t *testing.T) {
	kdf := testKDF(t)
	key, err := DeriveKey("correct horse battery", kdf)
	if err != nil {
		t.Fatal(err)
	}
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-test"
	secrets := Secrets{Version: 1, Instances: map[string]obelisk.InstanceCredentials{}}
	bundle, err := Pack(spec, secrets, key, kdf, 1)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := DeriveKey("nope", kdf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Unpack(bundle.Files, wrong); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("wrong key: err = %v, want ErrWrongPassphrase", err)
	}

	// Key id matches but ciphertext was swapped — mimics torn read across commits.
	tampered := map[string][]byte{}
	for k, v := range bundle.Files {
		tampered[k] = append([]byte(nil), v...)
	}
	tampered[FileSecrets][len(tampered[FileSecrets])-1] ^= 0xff
	if _, err := Unpack(tampered, key); !errors.Is(err, ErrBundleCorrupt) {
		t.Fatalf("tampered ciphertext: err = %v, want ErrBundleCorrupt", err)
	}
}

func TestPackUnpackRoundTrip(t *testing.T) {
	kdf := testKDF(t)
	key, err := DeriveKey("phrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	spec := obelisk.DefaultSpec("vps")
	spec.Machine.ID = "obelisk-roundtrip"
	secrets := Secrets{Version: 1, Instances: map[string]obelisk.InstanceCredentials{}}
	bundle, err := Pack(spec, secrets, key, kdf, 42)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unpack(bundle.Files, key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.Generation != 42 {
		t.Fatalf("generation = %d", got.Manifest.Generation)
	}
	if got.Spec.Machine.ID != spec.Machine.ID {
		t.Fatalf("machine id = %q", got.Spec.Machine.ID)
	}
}
