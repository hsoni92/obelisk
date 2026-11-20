package hostidentity

import (
	"os"
	"strings"
)

var dmiProductUUIDPath = "/sys/class/dmi/id/product_uuid"

func dmiProductUUID() (string, bool) {
	path := dmiProductUUIDPath
	if override := strings.TrimSpace(os.Getenv("OBELISK_DMI_PRODUCT_UUID_PATH")); override != "" {
		path = override
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	uuid := strings.ToLower(strings.TrimSpace(string(data)))
	if !validDMIUUID(uuid) {
		return "", false
	}
	return "dmi:" + uuid, true
}

func validDMIUUID(uuid string) bool {
	if uuid == "" {
		return false
	}
	switch uuid {
	case "ffffffff-ffff-ffff-ffff-ffffffffffff",
		"00000000-0000-0000-0000-000000000000",
		"not settable",
		"to be filled by o.e.m.":
		return false
	}
	return len(uuid) >= 8
}