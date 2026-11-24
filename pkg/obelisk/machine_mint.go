package obelisk

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const machineIDDerivationPrefix = "obelisk/v1/machine-id\x00"

// MintMachineID assigns a new appliance id. Defaults to RandomMachineID; the
// controller and CLI link internal/hostidentity to mint from host anchors.
var MintMachineID = RandomMachineID

// DeriveMachineIDFromAnchor maps a stable host anchor to a deterministic bucket-safe id.
func DeriveMachineIDFromAnchor(anchor string) string {
	sum := sha256.Sum256(append([]byte(machineIDDerivationPrefix), []byte(anchor)...))
	return "obelisk-" + hex.EncodeToString(sum[:6])
}

// RandomMachineID returns a DNS-safe S3 bucket name unrelated to the host.
func RandomMachineID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "obelisk-" + hex.EncodeToString(b)
}

// NewMachineID is an alias for RandomMachineID.
func NewMachineID() string {
	return RandomMachineID()
}
