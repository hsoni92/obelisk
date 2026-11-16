package obelisk

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Connections holds copy-ready endpoint hints for the admin UI.
type Connections struct {
	Instances []InstanceConnection `json:"instances,omitempty"`
	Notes     string               `json:"notes,omitempty"`
}

// InstanceConnection is one instance endpoint block.
type InstanceConnection struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	Postgres    string `json:"postgres,omitempty"`
	MySQL       string `json:"mysql,omitempty"`
	MongoDB     string `json:"mongodb,omitempty"`
	Redis       string `json:"redis,omitempty"`
	S3HTTP      string `json:"s3_http,omitempty"`
	QdrantHTTP  string `json:"qdrant_http,omitempty"`
	QdrantGRPC  string `json:"qdrant_grpc,omitempty"`
	SearXNGHTTP string `json:"searxng_http,omitempty"`
}

// PublicHost returns the hostname clients should use for native protocols.
func PublicHost() string {
	if h := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST")); h != "" {
		return h
	}
	return "localhost"
}

// BuildConnections produces connection strings for enabled running instances.
func BuildConnections(spec Spec, states map[string]string) Connections {
	host := PublicHost()
	out := Connections{}

	for _, inst := range spec.Instances {
		if !inst.Enabled || states[inst.StateKey()] != "running" {
			continue
		}
		ic := InstanceConnection{
			ID:          inst.ID,
			Type:        inst.Type,
			DisplayName: inst.DisplayName,
		}
		switch inst.Type {
		case "postgres", "mysql", "mongodb":
			user := inst.Credentials.Username
			if user == "" {
				user = "obelisk"
			}
			db := inst.Credentials.Database
			if db == "" {
				db = "obelisk"
			}
			pass := url.QueryEscape(inst.Credentials.Password)
			if pass == "" {
				pass = "SECRET"
			}
			switch inst.Type {
			case "postgres":
				ic.Postgres = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", user, pass, host, db)
			case "mysql":
				ic.MySQL = fmt.Sprintf("mysql://%s:%s@%s:3306/%s", user, pass, host, db)
			case "mongodb":
				ic.MongoDB = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin", user, pass, host, db)
			}
		case "redis":
			if inst.Credentials.Password != "" {
				ic.Redis = fmt.Sprintf("redis://:%s@%s:6379/0", url.QueryEscape(inst.Credentials.Password), host)
			} else {
				ic.Redis = fmt.Sprintf("redis://%s:6379/0", host)
			}
		case "rustfs":
			if p := strings.TrimSpace(os.Getenv("OBELISK_HTTP_PORT")); p != "" {
				ic.S3HTTP = fmt.Sprintf("http://%s:%s/s3", host, p)
			} else {
				ic.S3HTTP = fmt.Sprintf("http://%s/s3", host)
			}
		case "qdrant":
			ic.QdrantHTTP = fmt.Sprintf("http://%s:6333", host)
			ic.QdrantGRPC = fmt.Sprintf("%s:6334", host)
		case "searxng":
			ic.SearXNGHTTP = fmt.Sprintf("http://%s:8080", host)
		}
		out.Instances = append(out.Instances, ic)
	}
	return out
}