package hostidentity

import "github.com/hsoni92/obelisk/pkg/obelisk"

func init() {
	obelisk.MintMachineID = Mint
}
