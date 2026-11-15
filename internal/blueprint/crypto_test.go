package blueprint

import (
	"bytes"
	"testing"
)

func testKDF(t *testing.T) KDFParams {
	t.Helper()
	return KDFParams{
		Memory:      16,
		Iterations:  1,
		Parallelism: 1,
		Salt:        bytes.Repeat([]byte{7}, 16),
	}
}

func TestEncryptRoundTripAndFailures(t *testing.T) {
	kdf := testKDF(t)
	key, err := DeriveKey("correct horse battery", kdf)
	if err != nil {
		t.Fatal(err)
	}
	nonce, ct, err := Encrypt(key, []byte(`{"token":"secret"}`), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(key, nonce, ct, []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != `{"token":"secret"}` {
		t.Fatalf("plain = %s", plain)
	}

	wrong, err := DeriveKey("nope", kdf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(wrong, nonce, ct, []byte("aad")); err != ErrTamper {
		t.Fatalf("wrong key err = %v", err)
	}
	ct[len(ct)-1] ^= 0xff
	if _, err := Decrypt(key, nonce, ct, []byte("aad")); err != ErrTamper {
		t.Fatalf("tamper err = %v", err)
	}
	if KeyID(key) == KeyID(wrong) {
		t.Fatal("distinct keys produced the same key id")
	}
}

func TestPassphraseNotInError(t *testing.T) {
	kdf := testKDF(t)
	key, err := DeriveKey("super-secret-passphrase", kdf)
	if err != nil {
		t.Fatal(err)
	}
	nonce, ct, err := Encrypt(key, []byte("x"), []byte("aad"))
	if err != nil {
		t.Fatal(err)
	}
	other, _ := DeriveKey("other", kdf)
	_, err = Decrypt(other, nonce, ct, []byte("aad"))
	if err == nil || bytes.Contains([]byte(err.Error()), []byte("super-secret-passphrase")) {
		t.Fatalf("error leaked passphrase: %v", err)
	}
}
