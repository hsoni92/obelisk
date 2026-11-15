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
	var notes []string

	switch spec.Deployment.Profile {
	case "local":
		if spec.Deployment.Tailscale.Enabled {
			notes = append(notes, "Tailscale enabled: set OBELISK_PUBLIC_HOST to your tailnet machine name for remote clients.")
		} else {
			notes = append(notes, "Local profile: use port-forward or OBELISK_PUBLIC_HOST. Postgres/Redis are not via Kong.")
		}
	default:
		notes = append(notes, "VPS profile: open firewall for Postgres/Redis if clients are off-box.")
	}

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
			svc := inst.ReleaseName()
			switch inst.Type {
			case "postgres":
				ic.Postgres = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", user, pass, host, db)
				notes = append(notes, fmt.Sprintf("Postgres %q service: %s (port-forward 5432 if local)", inst.ID, svc))
			case "mysql":
				ic.MySQL = fmt.Sprintf("mysql://%s:%s@%s:3306/%s", user, pass, host, db)
				notes = append(notes, fmt.Sprintf("MySQL %q service: %s (port-forward 3306 if local)", inst.ID, svc))
			case "mongodb":
				ic.MongoDB = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin", user, pass, host, db)
				notes = append(notes, fmt.Sprintf("MongoDB %q service: %s (port-forward 27017 if local)", inst.ID, svc))
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
			svc := inst.ReleaseName()
			notes = append(notes, fmt.Sprintf("Qdrant %q: REST %s, gRPC %s (port-forward 6333/6334 if local); send api-key header when auth is enabled", inst.ID, svc, svc))
		case "searxng":
			ic.SearXNGHTTP = fmt.Sprintf("http://%s:8080", host)
			svc := inst.ReleaseName()
			notes = append(notes, fmt.Sprintf("SearXNG %q service: %s (port-forward 8080 if local)", inst.ID, svc))
		}
		out.Instances = append(out.Instances, ic)
	}
	if len(notes) > 0 {
		out.Notes = strings.Join(uniqueStrings(notes), " ")
	}
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}