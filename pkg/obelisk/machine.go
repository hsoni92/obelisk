package obelisk

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Machine holds stable appliance identity (backup bucket name, restore correlation).
type Machine struct {
	ID string `yaml:"id" json:"id"`
}

// Backup layout inside the machine bucket (operator does not configure these).
const (
	BackupConfigPrefix   = "config/"
	BackupDataPrefix     = "data/"
	BackupPreflightObject = "preflight/.write-test"
)

var machineIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,62}[a-z0-9]$`)

// EnsureMachineIdentity assigns a persistent id when missing. Returns true if a new id was set.
func EnsureMachineIdentity(spec *Spec) bool {
	id := strings.TrimSpace(spec.Machine.ID)
	if id != "" {
		spec.Machine.ID = id
		return false
	}
	spec.Machine.ID = NewMachineID()
	return true
}

// NewMachineID returns a DNS-safe S3 bucket name for this appliance.
func NewMachineID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "obelisk-" + hex.EncodeToString(b)
}

// ValidateMachineID checks the stored identity.
func ValidateMachineID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("machine.id is required")
	}
	if len(id) < 4 || len(id) > 63 {
		return fmt.Errorf("machine.id must be 4–63 characters for S3 bucket rules")
	}
	if !machineIDRe.MatchString(id) {
		return fmt.Errorf("machine.id must be lowercase letters, digits, and hyphens")
	}
	return nil
}

// BackupBucketName is the remote bucket for this appliance (equals machine.id).
func (s Spec) BackupBucketName() string {
	return strings.TrimSpace(s.Machine.ID)
}

// BackupRemoteReady is true when endpoint and machine identity are set.
func (s Spec) BackupRemoteReady() bool {
	return strings.TrimSpace(s.Backup.Remote.Endpoint) != "" && s.BackupBucketName() != ""
}

// BackupConfigObjectKey returns the object prefix for control-plane artifacts.
func BackupConfigObjectKey(name string) string {
	return BackupConfigPrefix + name
}

// BackupDataObjectKey returns the object prefix for a data-service backup.
func BackupDataObjectKey(serviceType, instanceID, name string) string {
	return fmt.Sprintf("%s%s/%s/%s", BackupDataPrefix, serviceType, instanceID, name)
}