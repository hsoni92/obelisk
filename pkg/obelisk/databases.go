package obelisk

import (
	"fmt"
	"regexp"
	"strings"
)

// BootstrapDatabaseName is created on first launch (POSTGRES_DB / MYSQL_DATABASE / etc.).
const BootstrapDatabaseName = "default"

var databaseNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// IsRelationalEngine reports types that support logical databases.
func IsRelationalEngine(typ string) bool {
	switch typ {
	case "postgres", "mysql", "mongodb":
		return true
	default:
		return false
	}
}

// ValidateDatabaseName checks a logical database name for engines that support it.
func ValidateDatabaseName(name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return fmt.Errorf("database name is required")
	}
	if name == BootstrapDatabaseName {
		return fmt.Errorf("use %q for the bootstrap database; add it via launch, not this list", BootstrapDatabaseName)
	}
	if !databaseNameRE.MatchString(name) {
		return fmt.Errorf("database name must match [a-z][a-z0-9_]{0,62}")
	}
	return nil
}

// BootstrapDatabase returns the primary database name for connection URLs and chart bootstrap.
func BootstrapDatabase(inst ServiceInstance) string {
	if !IsRelationalEngine(inst.Type) {
		return ""
	}
	db := strings.TrimSpace(inst.Credentials.Database)
	if db == "" {
		return BootstrapDatabaseName
	}
	return db
}

// DesiredDatabaseNames returns bootstrap + additional databases (deduped, ordered).
func DesiredDatabaseNames(inst ServiceInstance) []string {
	if !IsRelationalEngine(inst.Type) {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	add(BootstrapDatabase(inst))
	for _, name := range inst.Databases {
		add(name)
	}
	return out
}