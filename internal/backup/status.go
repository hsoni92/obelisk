package backup

import (
	"encoding/json"
	"os"
	"time"

	"github.com/hsoni92/obelisk/internal/config"
)

// DestinationRuntimeStatus is volatile per-destination health (backup-status.json).
type DestinationRuntimeStatus struct {
	LastTestAt      string `json:"last_test_at,omitempty"`
	LastTestOK      *bool  `json:"last_test_ok,omitempty"`
	LastTestMessage string `json:"last_test_message,omitempty"`
	LastUploadAt    string `json:"last_upload_at,omitempty"`
	LastInventoryAt string `json:"last_inventory_at,omitempty"`
	Bytes           int64  `json:"bytes,omitempty"`
	Objects         int    `json:"objects,omitempty"`
	LastError       string `json:"last_error,omitempty"`
}

// StatusFile tracks DR runtime state beside obelisk.yaml.
type StatusFile struct {
	LastRunAt string `json:"last_run_at,omitempty"`
	// Instances maps "<type>/<id>" to the RFC3339 time its data was last uploaded.
	Instances    map[string]string                   `json:"instances,omitempty"`
	Destinations map[string]DestinationRuntimeStatus `json:"destinations,omitempty"`
}

func statusPath(configPath string) string {
	return config.CredentialsDir(configPath) + "/backup-status.json"
}

func LoadStatus(configPath string) (StatusFile, error) {
	p := statusPath(configPath)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusFile{Destinations: map[string]DestinationRuntimeStatus{}}, nil
		}
		return StatusFile{}, err
	}
	var s StatusFile
	if err := json.Unmarshal(data, &s); err != nil {
		return StatusFile{}, err
	}
	if s.Destinations == nil {
		s.Destinations = map[string]DestinationRuntimeStatus{}
	}
	return s, nil
}

func SaveStatus(configPath string, s StatusFile) error {
	if s.Destinations == nil {
		s.Destinations = map[string]DestinationRuntimeStatus{}
	}
	dir := config.CredentialsDir(configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p := statusPath(configPath)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// RenameDestinationStatus moves runtime status from oldID to newID.
func RenameDestinationStatus(configPath, oldID, newID string) error {
	if oldID == newID {
		return nil
	}
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	st, ok := s.Destinations[oldID]
	if !ok {
		return nil
	}
	delete(s.Destinations, oldID)
	s.Destinations[newID] = st
	return SaveStatus(configPath, s)
}

// SwapDestinationStatus exchanges runtime status for two destination ids.
func SwapDestinationStatus(configPath, idA, idB string) error {
	if idA == idB {
		return nil
	}
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	a, okA := s.Destinations[idA]
	b, okB := s.Destinations[idB]
	if okA {
		s.Destinations[idB] = a
	} else {
		delete(s.Destinations, idB)
	}
	if okB {
		s.Destinations[idA] = b
	} else {
		delete(s.Destinations, idA)
	}
	return SaveStatus(configPath, s)
}

func UpdateDestinationTest(configPath, id string, ok bool, message string) error {
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	st := s.Destinations[id]
	now := time.Now().UTC().Format(time.RFC3339)
	st.LastTestAt = now
	st.LastTestOK = &ok
	st.LastTestMessage = message
	if !ok {
		st.LastError = message
	} else {
		st.LastError = ""
	}
	s.Destinations[id] = st
	return SaveStatus(configPath, s)
}

func UpdateDestinationUpload(configPath, id string) error {
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	st := s.Destinations[id]
	st.LastUploadAt = time.Now().UTC().Format(time.RFC3339)
	s.Destinations[id] = st
	return SaveStatus(configPath, s)
}

func UpdateLastRun(configPath string) error {
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	s.LastRunAt = time.Now().UTC().Format(time.RFC3339)
	return SaveStatus(configPath, s)
}

func UpdateDestinationInventory(configPath, id string, bytes int64, objects int) error {
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	st := s.Destinations[id]
	st.LastInventoryAt = time.Now().UTC().Format(time.RFC3339)
	st.Bytes = bytes
	st.Objects = objects
	s.Destinations[id] = st
	return SaveStatus(configPath, s)
}

// UpdateInstanceBackups records a successful data upload for each "<type>/<id>" key.
func UpdateInstanceBackups(configPath string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	s, err := LoadStatus(configPath)
	if err != nil {
		return err
	}
	if s.Instances == nil {
		s.Instances = map[string]string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, k := range keys {
		s.Instances[k] = now
	}
	return SaveStatus(configPath, s)
}
