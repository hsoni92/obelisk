package hostidentity

import (
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// Mint chooses a machine.id: persisted file, host-derived, or random.
func Mint() string {
	if id, ok := readPersistedID(); ok {
		return id
	}
	if anchor, ok := DiscoverAnchor(); ok {
		id := obelisk.DeriveMachineIDFromAnchor(anchor)
		if err := obelisk.ValidateMachineID(id); err == nil {
			_ = writePersistedID(id)
			return id
		}
	}
	id := obelisk.RandomMachineID()
	_ = writePersistedID(id)
	return id
}