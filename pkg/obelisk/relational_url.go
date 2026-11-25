package obelisk

import (
	"fmt"
	"net/url"
	"strings"
)

// DatabaseConnection is one named connection URL for a relational instance.
type DatabaseConnection struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// RelationalConnectionURL builds a driver URL for a logical database on an instance.
func RelationalConnectionURL(spec Spec, inst ServiceInstance, dbName string) string {
	if !IsRelationalEngine(inst.Type) {
		return ""
	}
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		return ""
	}
	user := strings.TrimSpace(inst.Credentials.Username)
	if user == "" {
		user = "obelisk"
	}
	pass := url.QueryEscape(inst.Credentials.Password)
	if pass == "" {
		pass = "SECRET"
	}
	host := PublicHost()
	vpsEdge := VPSEdge(spec)
	edgeHost := host
	if vpsEdge {
		edgeHost = PublicEdgeHost(inst)
	}
	switch inst.Type {
	case "postgres":
		if vpsEdge {
			return fmt.Sprintf(
				"postgresql://%s:%s@%s:%s/%s?%s",
				user, pass, edgeHost, NativeEdgeTLSPort(), dbName, VPSPostgresURLQuery,
			)
		}
		return fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", user, pass, host, dbName)
	case "mysql":
		if vpsEdge {
			return fmt.Sprintf(
				"mysql://%s:%s@tcp(%s:%s)/%s?%s",
				user, pass, edgeHost, NativeEdgeTLSPort(), dbName, VPSMySQLURLQuery,
			)
		}
		return fmt.Sprintf("mysql://%s:%s@%s:3306/%s", user, pass, host, dbName)
	case "mongodb":
		if vpsEdge {
			return fmt.Sprintf(
				"mongodb://%s:%s@%s:%s/%s?authSource=admin&tls=true&tlsAllowInvalidCertificates=true",
				user, pass, edgeHost, NativeEdgeTLSPort(), dbName,
			)
		}
		return fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin", user, pass, host, dbName)
	default:
		return ""
	}
}
