package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadRepoDotEnv reads $OBELISK_ROOT/.env (or repo root from cwd) into the environment.
// Variables that are already set in the process environment are left unchanged.
func LoadRepoDotEnv() {
	root := ResolveHelmRoot()
	LoadDotEnvFile(filepath.Join(root, ".env"))
}

// LoadDotEnvFile parses simple KEY=VALUE lines (optional quotes). Missing files are ignored.
func LoadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if cur := strings.TrimSpace(os.Getenv(key)); cur != "" {
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		_ = os.Setenv(key, val)
	}
}