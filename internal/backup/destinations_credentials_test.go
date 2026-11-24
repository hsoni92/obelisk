package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveDestinationsCredentialsUsesCredentialsDir(t *testing.T) {
	t.Setenv("OBELISK_CREDENTIALS_DIR", t.TempDir())
	cfg := filepath.Join(t.TempDir(), "obelisk.yaml")
	if err := SaveDestinationsCredentials(cfg, DestinationsCredentialsFile{
		Destinations: map[string]RemoteCredentials{
			"primary": {AccessKeyID: "ak", SecretAccessKey: "sk"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	path := destinationsCredentialsPath(cfg)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDestinationsCredentials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Destinations["primary"].AccessKeyID != "ak" {
		t.Fatalf("got %#v", loaded)
	}
}
