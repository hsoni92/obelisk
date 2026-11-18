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
	Databases   []DatabaseConnection `json:"databases,omitempty"`
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
	vpsEdge := VPSEdge(spec)
	out := Connections{}
	if vpsEdge {
		out.Notes = "VPS edge: HTTP APIs on port " + AdminHTTPPortString() + " (per-instance hostnames). Native Redis/Postgres/MySQL/Mongo use TLS on port " + NativeEdgeTLSPort() + " (per-instance hostnames; Kong SNI passthrough). Trust Obelisk's auto-generated edge certificate presented by the database service. Obelisk never binds host ports 80 or 443."
	}

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
		if vpsEdge {
			if h := PublicEdgeHost(inst); h != "" {
				ic.PublicPath = h
			}
		} else if p := PublicHTTPPath(inst); p != "" {
			ic.PublicPath = p
		}

		switch inst.Type {
		case "postgres", "mysql", "mongodb":
			user := inst.Credentials.Username
			if user == "" {
				user = "obelisk"
			}
			db := BootstrapDatabase(inst)
			pass := url.QueryEscape(inst.Credentials.Password)
			if pass == "" {
				pass = "SECRET"
			}
			edgeHost := host
			if vpsEdge {
				edgeHost = PublicEdgeHost(inst)
			}
			switch inst.Type {
			case "postgres":
				if vpsEdge {
					ic.Postgres = fmt.Sprintf(
						"postgresql://%s:%s@%s:%s/%s?sslmode=require",
						user, pass, edgeHost, NativeEdgeTLSPort(), db,
					)
				} else {
					ic.Postgres = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", user, pass, host, db)
				}
			case "mysql":
				if vpsEdge {
					ic.MySQL = fmt.Sprintf(
						"mysql://%s:%s@tcp(%s:%s)/%s?tls=true",
						user, pass, edgeHost, NativeEdgeTLSPort(), db,
					)
				} else {
					ic.MySQL = fmt.Sprintf("mysql://%s:%s@%s:3306/%s", user, pass, host, db)
				}
			case "mongodb":
				if vpsEdge {
					ic.MongoDB = fmt.Sprintf(
						"mongodb://%s:%s@%s:%s/%s?authSource=admin&tls=true",
						user, pass, edgeHost, NativeEdgeTLSPort(), db,
					)
				} else {
					ic.MongoDB = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=admin", user, pass, host, db)
				}
			}
		case "redis":
			if vpsEdge {
				edgeHost := PublicEdgeHost(inst)
				if inst.Credentials.Password != "" {
					ic.Redis = fmt.Sprintf("rediss://:%s@%s:%s/0", url.QueryEscape(inst.Credentials.Password), edgeHost, NativeEdgeTLSPort())
				} else {
					ic.Redis = fmt.Sprintf("rediss://%s:%s/0", edgeHost, NativeEdgeTLSPort())
				}
			} else if inst.Credentials.Password != "" {
				ic.Redis = fmt.Sprintf("redis://:%s@%s:6379/0", url.QueryEscape(inst.Credentials.Password), host)
			} else {
				ic.Redis = fmt.Sprintf("redis://%s:6379/0", host)
			}
		case "rustfs":
			if vpsEdge {
				ic.S3HTTP = PublicS3Endpoint(inst)
			} else {
				ic.S3HTTP = PublicS3Endpoint(inst)
			}
		case "qdrant":
			if vpsEdge {
				ic.QdrantHTTP = PublicHTTPEdgeURL(inst)
				ic.QdrantGRPC = ""
			} else {
				ic.QdrantHTTP = fmt.Sprintf("http://%s:6333", host)
				ic.QdrantGRPC = fmt.Sprintf("%s:6334", host)
			}
		case "searxng":
			if vpsEdge {
				ic.SearXNGHTTP = PublicHTTPEdgeURL(inst)
			} else {
				ic.SearXNGHTTP = fmt.Sprintf("http://%s:8080", host)
			}
		}
		if IsRelationalEngine(inst.Type) {
			for _, name := range DesiredDatabaseNames(inst) {
				if u := RelationalConnectionURL(spec, inst, name); u != "" {
					ic.Databases = append(ic.Databases, DatabaseConnection{Name: name, URL: u})
				}
			}
		}
		ic.Use = instanceUse(spec, inst, ic)
		out.Instances = append(out.Instances, ic)
	}
	return out
}

// instanceUse returns one admin-shareable credential block per service type.
func instanceUse(spec Spec, inst ServiceInstance, ic InstanceConnection) string {
	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		return relationalUseBlock(spec, inst, ic)
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
			if ic.QdrantGRPC == "" {
				return ic.QdrantHTTP
			}
			return fmt.Sprintf("%s\n%s", ic.QdrantHTTP, ic.QdrantGRPC)
		}
		if ic.QdrantGRPC == "" {
			return fmt.Sprintf("QDRANT_URL=%s\nQDRANT_API_KEY=%s", ic.QdrantHTTP, key)
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

func relationalUseBlock(spec Spec, inst ServiceInstance, ic InstanceConnection) string {
	var primary string
	switch inst.Type {
	case "postgres":
		primary = ic.Postgres
	case "mysql":
		primary = ic.MySQL
	case "mongodb":
		primary = ic.MongoDB
	}
	if primary == "" {
		return ""
	}
	bootstrap := BootstrapDatabase(inst)
	lines := []string{
		fmt.Sprintf("# database %s (bootstrap)", bootstrap),
		primary,
	}
	for _, name := range inst.Databases {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" || name == bootstrap {
			continue
		}
		if u := RelationalConnectionURL(spec, inst, name); u != "" {
			lines = append(lines, fmt.Sprintf("# database %s", name), u)
		}
	}
	return strings.Join(lines, "\n")
}