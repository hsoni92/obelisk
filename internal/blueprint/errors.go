package blueprint

import "errors"

var (
	ErrImportRequired      = errors.New("import required: remote blueprint generation is ahead of this appliance")
	ErrLocked              = errors.New("blueprint crypto is locked")
	ErrWrongPassphrase     = errors.New("wrong passphrase")
	ErrTamper              = errors.New("blueprint ciphertext failed authentication")
	ErrEmptyRepo           = errors.New("blueprint bundle not in repository")
	ErrMachineChoice       = errors.New("machine id differs from blueprint; choose adopt or keep")
	ErrClientNotConfigured = errors.New("GitHub OAuth client is not configured")
	ErrPending             = errors.New("authorization pending")
	ErrOnboardingRequired  = errors.New("blueprint onboarding is required")
)

const (
	FileManifest = "manifest.json"
	FilePublic   = "obelisk.public.yaml"
	FileSecrets  = "secrets.enc"

	SchemaVersion = 1
	Algorithm     = "xchacha20poly1305"
	KDFName       = "argon2id"

	// Production Argon2id parameters (OWASP). Tests may use smaller params
	// stored in the manifest; decrypt always honors the manifest.
	ArgonMemoryKiB  uint32 = 64 * 1024
	ArgonIterations uint32 = 3
	ArgonThreads    uint8  = 4
	ArgonKeyLen            = 32
	ArgonSaltLen           = 16
)

// CookieName is the admin UI session cookie set after GitHub login.
const CookieName = "obelisk_session"

// TokenSecretName is the Kubernetes Secret (and local file stem) for OAuth tokens.
const TokenSecretName = "obelisk-blueprint-github"
