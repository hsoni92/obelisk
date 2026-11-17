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

// InstanceConnection is one instance endpoint block for the admin console.
// Use is the single copy-paste credential clients need (GitHub session required via GET /connections).
type InstanceConnection struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	Use         string `json:"use,omitempty"`
	Postgres    string `json:"postgres,omitempty"`
	MySQL       string `json:"mysql,omitempty"`
	MongoDB     string `json:"mongodb,omitempty"`
	Redis       string `json:"redis,omitempty"`
	S3HTTP      string `json:"s3_http,omitempty"`
	PublicPath  string `json:"public_path,omitempty"`
	RouteID     string `json:"route_id,omitempty"`
	QdrantHTTP  string `json:"qdrant_http,omitempty"`
	QdrantGRPC  string `json:"qdrant_grpc,omitempty"`
	SearXNGHTTP string `json:"searxng_http,omitempty"`
}

// PublicHost returns the hostname clients should use for native protocols.
func PublicHost() string {
	if h := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_HOST")); h != "" {
		return h
	}
	if u := strings.TrimSpace(os.Getenv("OBELISK_PUBLIC_URL")); u != "" {
		if parsed, err := url.Parse(u); err == nil {
			if h := parsed.Hostname(); h != "" {
				return h
			}
		}
	}
	return "localhost"
}

// BuildConnections produces shareable credentials for enabled instances.
// states is reserved for future reachability hints; URLs are built from desired state.
func BuildConnections(spec Spec, states map[string]string) Connections {
	_ = states
	host := PublicHost()
	out := Connections{}

	for _, inst := range spec.Instances {
		if !inst.Enabled {
			continue
		}
		ic := InstanceConnection{
			ID:          inst.ID,
			Type:        inst.Type,
			DisplayName: inst.DisplayName,
			RouteID:     inst.ID,
		}
		switch inst.Type {
		case "rustfs":
			if h := PublicS3Host(inst); h != "" {
				ic.PublicPath = h
			}
		default:
			if p := PublicHTTPPath(inst); p != "" {
				ic.PublicPath = p
			}
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
			ic.S3HTTP = PublicS3Endpoint(inst)
		case "qdrant":
			ic.QdrantHTTP = fmt.Sprintf("http://%s:6333", host)
			ic.QdrantGRPC = fmt.Sprintf("%s:6334", host)
		case "searxng":
			ic.SearXNGHTTP = fmt.Sprintf("http://%s:8080", host)
		}
		ic.Use = instanceUse(inst, ic)
		out.Instances = append(out.Instances, ic)
	}
	return out
}

// instanceUse returns one admin-shareable credential block per service type.
func instanceUse(inst ServiceInstance, ic InstanceConnection) string {
	switch inst.Type {
	case "postgres":
		return ic.Postgres
	case "mysql":
		return ic.MySQL
	case "mongodb":
		return ic.MongoDB
	case "redis":
		return ic.Redis
	case "rustfs":
		if ic.S3HTTP == "" {
			return ""
		}
		ak := strings.TrimSpace(inst.Credentials.AccessKey)
		sk := strings.TrimSpace(inst.Credentials.SecretKey)
		lines := []string{fmt.Sprintf("S3_ENDPOINT=%s", ic.S3HTTP)}
		if ak != "" {
			lines = append(lines, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", ak))
		}
		if sk != "" {
			lines = append(lines, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", sk))
		}
		if len(lines) == 1 {
			return ic.S3HTTP
		}
		return strings.Join(lines, "\n")
	case "qdrant":
		key := strings.TrimSpace(inst.Credentials.AccessKey)
		if ic.QdrantHTTP == "" {
			return ""
		}
		if key == "" {
			return fmt.Sprintf("%s\n%s", ic.QdrantHTTP, ic.QdrantGRPC)
		}
		return fmt.Sprintf(
			"QDRANT_URL=%s\nQDRANT_GRPC=%s\nQDRANT_API_KEY=%s",
			ic.QdrantHTTP, ic.QdrantGRPC, key,
		)
	case "searxng":
		sk := strings.TrimSpace(inst.Credentials.SecretKey)
		if ic.SearXNGHTTP == "" {
			return ""
		}
		if sk == "" {
			return ic.SearXNGHTTP
		}
		return fmt.Sprintf("%s\nsecret-key: %s", ic.SearXNGHTTP, sk)
	default:
		return ""
	}
}