package blueprint

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// KDFParams are stored in the manifest and on the spec. The salt is not secret.
type KDFParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	Salt        []byte
}

// DefaultKDFParams returns production Argon2id parameters and a fresh salt.
func DefaultKDFParams() (KDFParams, error) {
	salt := make([]byte, ArgonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return KDFParams{}, err
	}
	return KDFParams{
		Memory:      ArgonMemoryKiB,
		Iterations:  ArgonIterations,
		Parallelism: ArgonThreads,
		Salt:        salt,
	}, nil
}

// DeriveKey runs Argon2id. The passphrase is not retained.
func DeriveKey(passphrase string, p KDFParams) ([]byte, error) {
	if len(p.Salt) < 8 {
		return nil, fmt.Errorf("kdf salt is missing")
	}
	if p.Memory == 0 || p.Iterations == 0 || p.Parallelism == 0 {
		return nil, fmt.Errorf("kdf parameters are incomplete")
	}
	return argon2.IDKey([]byte(passphrase), p.Salt, p.Iterations, p.Memory, p.Parallelism, ArgonKeyLen), nil
}

// KeyID is the non-secret fingerprint of a derived key (envelope id).
func KeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:8])
}

// Encrypt seals plaintext with XChaCha20-Poly1305. aad binds the manifest.
func Encrypt(key, plaintext, aad []byte) (nonce, ciphertext []byte, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = aead.Seal(nil, nonce, plaintext, aad)
	return nonce, ciphertext, nil
}

// Decrypt opens ciphertext. A wrong key or edited bytes returns ErrTamper or ErrWrongPassphrase.
func Decrypt(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, ErrTamper
	}
	plain, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrTamper
	}
	return plain, nil
}
