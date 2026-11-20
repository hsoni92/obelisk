package hostidentity

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// ApplianceIDPath is where a minted machine.id is stored across config regen.
func ApplianceIDPath() string {
	if p := strings.TrimSpace(os.Getenv("OBELISK_APPLIANCE_ID_FILE")); p != "" {
		return filepath.Clean(p)
	}
	if d := strings.TrimSpace(os.Getenv("OBELISK_CREDENTIALS_DIR")); d != "" {
		return filepath.Join(filepath.Dir(filepath.Clean(d)), "appliance-id")
	}
	if c := strings.TrimSpace(os.Getenv("OBELISK_CONFIG")); c != "" {
		return filepath.Join(filepath.Dir(filepath.Clean(c)), "appliance-id")
	}
	return "/var/lib/obelisk/appliance-id"
}

func readPersistedID() (string, bool) {
	data, err := os.ReadFile(ApplianceIDPath())
	if err != nil {
		return "", false
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", false
	}
	if err := obelisk.ValidateMachineID(id); err != nil {
		return "", false
	}
	return id, true
}

// EnsurePersistedID writes appliance-id when missing (yaml is source of truth).
func EnsurePersistedID(id string) {
	if _, ok := readPersistedID(); ok {
		return
	}
	_ = writePersistedID(id)
}

func writePersistedID(id string) error {
	id = strings.TrimSpace(id)
	if err := obelisk.ValidateMachineID(id); err != nil {
		return err
	}
	path := ApplianceIDPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(id+"\n"), 0o600)
}