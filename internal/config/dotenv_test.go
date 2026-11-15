package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("# comment\nFOO=bar\nEXISTING=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXISTING", "kept")

	LoadDotEnvFile(path)
	if got := os.Getenv("FOO"); got != "bar" {
		t.Fatalf("FOO=%q want bar", got)
	}
	if got := os.Getenv("EXISTING"); got != "kept" {
		t.Fatalf("EXISTING=%q want kept", got)
	}
}