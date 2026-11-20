package backup

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hsoni92/obelisk/internal/config"
)

// DestinationsCredentialsFile stores per-destination API keys beside obelisk.yaml.
type DestinationsCredentialsFile struct {
	Destinations map[string]RemoteCredentials `json:"destinations,omitempty"`
}

func destinationsCredentialsPath(configPath string) string {
	return config.CredentialsDir(configPath) + "/backup-destinations.credentials.json"
}

func LoadDestinationsCredentials(configPath string) (DestinationsCredentialsFile, error) {
	p := destinationsCredentialsPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DestinationsCredentialsFile{Destinations: map[string]RemoteCredentials{}}, nil
		}
		return DestinationsCredentialsFile{}, err
	}
	var c DestinationsCredentialsFile
	if err := json.Unmarshal(data, &c); err != nil {
		return DestinationsCredentialsFile{}, fmt.Errorf("parse backup destinations credentials: %w", err)
	}
	if c.Destinations == nil {
		c.Destinations = map[string]RemoteCredentials{}
	}
	return c, nil
}

func SaveDestinationsCredentials(configPath string, c DestinationsCredentialsFile) error {
	if c.Destinations == nil {
		c.Destinations = map[string]RemoteCredentials{}
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	p := destinationsCredentialsPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// MigrateLegacyCredentials copies single-target keys into the destinations map when missing.
func MigrateLegacyCredentials(configPath, destinationID string) error {
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return err
	}
	if _, ok := file.Destinations[destinationID]; ok {
		c := file.Destinations[destinationID]
		if c.AccessKeyID != "" && c.SecretAccessKey != "" {
			return nil
		}
	}
	legacy, ok, err := LoadCredentials(configPath)
	if err != nil || !ok {
		return err
	}
	file.Destinations[destinationID] = legacy
	return SaveDestinationsCredentials(configPath, file)
}

func DestinationCredentialsConfigured(configPath, destinationID string) (bool, error) {
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return false, err
	}
	c, ok := file.Destinations[destinationID]
	if !ok {
		return false, nil
	}
	return c.AccessKeyID != "" && c.SecretAccessKey != "", nil
}

func LoadDestinationCredentials(configPath, destinationID string) (RemoteCredentials, bool, error) {
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return RemoteCredentials{}, false, err
	}
	c, ok := file.Destinations[destinationID]
	if !ok || c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return RemoteCredentials{}, false, nil
	}
	return c, true, nil
}

func SaveDestinationCredentials(configPath, destinationID string, creds RemoteCredentials) error {
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return fmt.Errorf("access_key_id and secret_access_key are required")
	}
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return err
	}
	file.Destinations[destinationID] = creds
	return SaveDestinationsCredentials(configPath, file)
}

func DeleteDestinationCredentials(configPath, destinationID string) error {
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return err
	}
	delete(file.Destinations, destinationID)
	return SaveDestinationsCredentials(configPath, file)
}

// SwapDestinationCredentials exchanges stored API keys for two destination ids.
func SwapDestinationCredentials(configPath, idA, idB string) error {
	if idA == idB {
		return nil
	}
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return err
	}
	a, okA := file.Destinations[idA]
	b, okB := file.Destinations[idB]
	if okA {
		file.Destinations[idB] = a
	} else {
		delete(file.Destinations, idB)
	}
	if okB {
		file.Destinations[idA] = b
	} else {
		delete(file.Destinations, idA)
	}
	return SaveDestinationsCredentials(configPath, file)
}

// AnyCredentialsConfigured is true when legacy or any destination has keys.
func AnyCredentialsConfigured(configPath string) (bool, error) {
	ok, err := CredentialsConfigured(configPath)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	file, err := LoadDestinationsCredentials(configPath)
	if err != nil {
		return false, err
	}
	for _, c := range file.Destinations {
		if c.AccessKeyID != "" && c.SecretAccessKey != "" {
			return true, nil
		}
	}
	return false, nil
}