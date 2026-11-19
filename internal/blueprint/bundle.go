package blueprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hsoni92/obelisk/pkg/obelisk"
	"gopkg.in/yaml.v3"
)

// Manifest is the cleartext index committed next to the ciphertext.
type Manifest struct {
	BlueprintSchemaVersion int    `json:"blueprint_schema_version"`
	APIVersion             string `json:"apiVersion"`
	Generation             int    `json:"generation"`
	MachineID              string `json:"machine_id"`
	Algorithm              string `json:"algorithm"`
	KDF                    string `json:"kdf"`
	KDFMemory              uint32 `json:"kdf_memory"`
	KDFIterations          uint32 `json:"kdf_iterations"`
	KDFParallelism         uint8  `json:"kdf_parallelism"`
	KDFSalt                string `json:"kdf_salt"`
	KeyID                  string `json:"key_id"`
	Nonce                  string `json:"nonce"`
	ContentSHA256          string `json:"content_sha256"`
	CommittedAt            string `json:"committed_at"`
}

// Secrets is the JSON document sealed inside secrets.enc.
type Secrets struct {
	Version            int                                    `json:"version"`
	Instances          map[string]obelisk.InstanceCredentials `json:"instances,omitempty"`
	BackupDestinations json.RawMessage                        `json:"backup_destinations,omitempty"`
	BackupLegacy       json.RawMessage                        `json:"backup_legacy,omitempty"`
	OpenRouter         json.RawMessage                        `json:"openrouter,omitempty"`
	Mail               json.RawMessage                        `json:"mail,omitempty"`
}

// Bundle is a packed v1 blueprint.
type Bundle struct {
	Manifest Manifest
	Files    map[string][]byte
	Spec     obelisk.Spec
	Secrets  Secrets
}

// Pack encrypts secrets and returns the three repo files.
func Pack(spec obelisk.Spec, secrets Secrets, key []byte, kdf KDFParams, generation int) (Bundle, error) {
	if generation < 1 {
		return Bundle{}, fmt.Errorf("generation must be >= 1")
	}
	secrets.Version = 1
	if secrets.Instances == nil {
		secrets.Instances = map[string]obelisk.InstanceCredentials{}
	}
	canonical, err := canonicalBytes(spec, secrets)
	if err != nil {
		return Bundle{}, err
	}
	sum := sha256.Sum256(canonical)
	contentHash := hex.EncodeToString(sum[:])

	pub := displaySpec(spec, generation)
	publicYAML, err := yaml.Marshal(pub)
	if err != nil {
		return Bundle{}, err
	}
	if err := ScanPublicYAML(publicYAML); err != nil {
		return Bundle{}, err
	}
	secretJSON, err := json.Marshal(secrets)
	if err != nil {
		return Bundle{}, err
	}

	manifest := Manifest{
		BlueprintSchemaVersion: SchemaVersion,
		APIVersion:             "obelisk.dev/blueprint/v1",
		Generation:             generation,
		MachineID:              spec.Machine.ID,
		Algorithm:              Algorithm,
		KDF:                    KDFName,
		KDFMemory:              kdf.Memory,
		KDFIterations:          kdf.Iterations,
		KDFParallelism:         kdf.Parallelism,
		KDFSalt:                base64.StdEncoding.EncodeToString(kdf.Salt),
		KeyID:                  KeyID(key),
		ContentSHA256:          contentHash,
		CommittedAt:            time.Now().UTC().Format(time.RFC3339),
	}
	nonce, ct, err := Encrypt(key, secretJSON, aad(manifest))
	if err != nil {
		return Bundle{}, err
	}
	manifest.Nonce = base64.StdEncoding.EncodeToString(nonce)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{
		Manifest: manifest,
		Spec:     pub,
		Secrets:  secrets,
		Files: map[string][]byte{
			FileManifest: manifestJSON,
			FilePublic:   publicYAML,
			FileSecrets:  ct,
		},
	}, nil
}

// Unpack decrypts a bundle. Wrong key id → ErrWrongPassphrase. Key id matches but
// AEAD fails → ErrBundleCorrupt (files from different commits or hand-edited manifest).
func Unpack(files map[string][]byte, key []byte) (Bundle, error) {
	rawManifest := files[FileManifest]
	publicYAML := files[FilePublic]
	ct := files[FileSecrets]
	if len(rawManifest) == 0 || len(publicYAML) == 0 || len(ct) == 0 {
		return Bundle{}, ErrEmptyRepo
	}
	var manifest Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return Bundle{}, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.BlueprintSchemaVersion != SchemaVersion {
		return Bundle{}, fmt.Errorf("unsupported blueprint_schema_version %d", manifest.BlueprintSchemaVersion)
	}
	if manifest.KeyID != "" && KeyID(key) != manifest.KeyID {
		return Bundle{}, ErrWrongPassphrase
	}
	nonce, err := base64.StdEncoding.DecodeString(manifest.Nonce)
	if err != nil {
		return Bundle{}, ErrTamper
	}
	plain, err := Decrypt(key, nonce, ct, aad(manifest))
	if err != nil {
		if manifest.KeyID != "" && KeyID(key) == manifest.KeyID {
			return Bundle{}, ErrBundleCorrupt
		}
		return Bundle{}, ErrWrongPassphrase
	}
	var secrets Secrets
	if err := json.Unmarshal(plain, &secrets); err != nil {
		return Bundle{}, fmt.Errorf("parse secrets: %w", err)
	}
	var spec obelisk.Spec
	if err := yaml.Unmarshal(publicYAML, &spec); err != nil {
		return Bundle{}, fmt.Errorf("parse public yaml: %w", err)
	}
	got, err := canonicalBytes(spec, secrets)
	if err != nil {
		return Bundle{}, err
	}
	sum := sha256.Sum256(got)
	if hex.EncodeToString(sum[:]) != manifest.ContentSHA256 {
		return Bundle{}, ErrBundleCorrupt
	}
	return Bundle{Manifest: manifest, Files: files, Spec: spec, Secrets: secrets}, nil
}

func aad(m Manifest) []byte {
	return []byte(fmt.Sprintf("%d|%s|%d|%s", m.BlueprintSchemaVersion, m.MachineID, m.Generation, m.ContentSHA256))
}

func displaySpec(spec obelisk.Spec, generation int) obelisk.Spec {
	pub := spec.RedactedSpec()
	pub.Blueprint.GitHub.Generation = generation
	pub.Blueprint.GitHub.SyncStatus = obelisk.SyncSynced
	pub.Blueprint.GitHub.LastPushError = ""
	pub.Blueprint.GitHub.CryptoUnlocked = false
	pub.Blueprint.GitHub.LastCommitSHA = ""
	return pub
}

// canonicalBytes hashes stable desired state. Sync bookkeeping is excluded so
// a metadata-only save does not look like a new blueprint.
// ContentHash is the stable hash stored in the manifest. Sync bookkeeping is excluded.
func ContentHash(spec obelisk.Spec, secrets Secrets) (string, error) {
	b, err := canonicalBytes(spec, secrets)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalBytes(spec obelisk.Spec, secrets Secrets) ([]byte, error) {
	pub := spec.RedactedSpec()
	pub.Blueprint.GitHub.Generation = 0
	pub.Blueprint.GitHub.LastSyncAt = ""
	pub.Blueprint.GitHub.LastCommitSHA = ""
	pub.Blueprint.GitHub.SyncStatus = ""
	pub.Blueprint.GitHub.CryptoUnlocked = false
	pub.Blueprint.GitHub.LastPushError = ""
	y, err := yaml.Marshal(pub)
	if err != nil {
		return nil, err
	}
	secrets.Version = 1
	s, err := json.Marshal(secrets)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(y)
	buf.WriteByte(0)
	buf.Write(s)
	return buf.Bytes(), nil
}

// KDFFromManifest reads the parameters needed to re-derive a key.
func KDFFromManifest(m Manifest) (KDFParams, error) {
	salt, err := base64.StdEncoding.DecodeString(m.KDFSalt)
	if err != nil {
		return KDFParams{}, fmt.Errorf("manifest kdf salt: %w", err)
	}
	return KDFParams{
		Memory:      m.KDFMemory,
		Iterations:  m.KDFIterations,
		Parallelism: m.KDFParallelism,
		Salt:        salt,
	}, nil
}
