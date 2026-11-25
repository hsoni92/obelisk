package obelisk

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Connections holds copy-ready endpoint hints for the admin UI.
type Connections struct {
	Instances      []InstanceConnection `json:"instances,omitempty"`
	Notes          string               `json:"notes,omitempty"`
	InClusterNotes string               `json:"in_cluster_notes,omitempty"`
}

// InstanceConnection is one instance endpoint block for the admin console.
// Use is the single copy-paste credential clients need (GitHub session required via GET /connections).
type InstanceConnection struct {
	ID                 string               `json:"id"`
	Type               string               `json:"type"`
	DisplayName        string               `json:"display_name,omitempty"`
	Use                string               `json:"use,omitempty"`
	TLS                *ConnectionTLS       `json:"tls,omitempty"`
	Postgres           string               `json:"postgres,omitempty"`
	MySQL              string               `json:"mysql,omitempty"`
	MongoDB            string               `json:"mongodb,omitempty"`
	Redis              string               `json:"redis,omitempty"`
	S3HTTP             string               `json:"s3_http,omitempty"`
	PublicPath         string               `json:"public_path,omitempty"`
	RouteID            string               `json:"route_id,omitempty"`
	QdrantHTTP         string               `json:"qdrant_http,omitempty"`
	QdrantGRPC         string               `json:"qdrant_grpc,omitempty"`
	SearXNGHTTP        string               `json:"searxng_http,omitempty"`
	UnimeterAPI        string               `json:"unimeter_api,omitempty"`
	UnimeterHealthHTTP string               `json:"unimeter_health_http,omitempty"`
	BunqueueTCP        string               `json:"bunqueue_tcp,omitempty"`
	BunqueueHealthHTTP string               `json:"bunqueue_health_http,omitempty"`
	BrowserlessHTTP    string               `json:"browserless_http,omitempty"`
	BrowserlessWS      string               `json:"browserless_ws,omitempty"`
	KafkaBootstrap     string               `json:"kafka_bootstrap,omitempty"`
	SchemaRegistryHTTP string               `json:"schema_registry_http,omitempty"`
	Databases          []DatabaseConnection `json:"databases,omitempty"`
	Origin             string               `json:"origin,omitempty"` // local | federated
	PeerID             string               `json:"peer_id,omitempty"`
	Reachable          *bool                `json:"reachable,omitempty"`
	// InCluster holds URLs for workloads on the same Kubernetes cluster.
	InCluster *ConnectionBlock `json:"in_cluster,omitempty"`
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
		out.Notes = "VPS edge: HTTP APIs on port " + AdminHTTPPortString() + " (per-instance hostnames). Native Redis/Postgres/MySQL/Mongo use TLS on port " + NativeEdgeTLSPort() + " (per-instance hostnames; nginx stream SNI passthrough). Trust Obelisk's auto-generated edge certificate presented by the database service. Postgres 17+: add sslnegotiation=direct to the URL. MySQL: open TLS to the edge host before the MySQL handshake (SNI); default drivers that upgrade after the server greeting will not work on :9443. Obelisk never binds host ports 80 or 443."
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
						"postgresql://%s:%s@%s:%s/%s?%s",
						user, pass, edgeHost, NativeEdgeTLSPort(), db, VPSPostgresURLQuery,
					)
				} else {
					ic.Postgres = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", user, pass, host, db)
				}
			case "mysql":
				if vpsEdge {
					ic.MySQL = fmt.Sprintf(
						"mysql://%s:%s@tcp(%s:%s)/%s?%s",
						user, pass, edgeHost, NativeEdgeTLSPort(), db, VPSMySQLURLQuery,
					)
				} else {
					ic.MySQL = fmt.Sprintf("mysql://%s:%s@%s:3306/%s", user, pass, host, db)
				}
			case "mongodb":
				if vpsEdge {
					ic.MongoDB = fmt.Sprintf(
						"mongodb://%s:%s@%s:%s/%s?authSource=admin&tls=true&tlsAllowInvalidCertificates=true",
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
		case "redpanda":
			bootstrapHost := host
			if vpsEdge {
				bootstrapHost = PublicEdgeHost(inst)
			}
			ic.KafkaBootstrap = fmt.Sprintf("%s:9092", bootstrapHost)
			ic.SchemaRegistryHTTP = schemaRegistryPublicURL(spec, inst, host)
		case "unimeter":
			apiHost := host
			healthHost := host
			if vpsEdge {
				apiHost = PublicEdgeHost(inst)
				healthHost = PublicEdgeHost(inst)
			} else if h := PublicEdgeHost(inst); h != "" && strings.TrimSpace(os.Getenv("OBELISK_DEV")) != "" {
				apiHost = h
				healthHost = h
			}
			ic.UnimeterAPI = fmt.Sprintf("%s:7001", apiHost)
			ic.UnimeterHealthHTTP = fmt.Sprintf("http://%s:9090/health", healthHost)
		case "browserless":
			if vpsEdge {
				ic.BrowserlessHTTP = PublicHTTPEdgeURL(inst)
				ic.BrowserlessWS = strings.Replace(ic.BrowserlessHTTP, "http://", "ws://", 1)
			} else {
				ic.BrowserlessHTTP = fmt.Sprintf("http://%s:3000", host)
				ic.BrowserlessWS = fmt.Sprintf("ws://%s:3000", host)
			}
		case "bunqueue":
			queueHost := host
			healthHost := host
			if vpsEdge {
				queueHost = PublicEdgeHost(inst)
				healthHost = PublicEdgeHost(inst)
				if u := PublicHTTPEdgeURL(inst); u != "" {
					ic.BunqueueHealthHTTP = strings.TrimSuffix(u, "/") + "/health"
				}
			} else if h := PublicEdgeHost(inst); h != "" && strings.TrimSpace(os.Getenv("OBELISK_DEV")) != "" {
				queueHost = h
				healthHost = h
			}
			ic.BunqueueTCP = fmt.Sprintf("%s:6789", queueHost)
			if ic.BunqueueHealthHTTP == "" {
				ic.BunqueueHealthHTTP = fmt.Sprintf("http://%s:6790/health", healthHost)
			}
		}
		if IsRelationalEngine(inst.Type) {
			for _, name := range DesiredDatabaseNames(inst) {
				if u := RelationalConnectionURL(spec, inst, name); u != "" {
					ic.Databases = append(ic.Databases, DatabaseConnection{Name: name, URL: u})
				}
			}
		}
		if vpsEdge {
			ic.TLS = edgeConnectionTLS(inst)
		}
		ic.Use = instanceUse(spec, inst, ic)
		attachInClusterBlock(&ic, spec, inst, ObeliskNamespace())
		out.Instances = append(out.Instances, ic)
	}
	out.InClusterNotes = InClusterConnectionNotes(ObeliskNamespace())
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
	case "redpanda":
		if ic.KafkaBootstrap == "" {
			return ""
		}
		lines := []string{
			fmt.Sprintf("KAFKA_BOOTSTRAP_SERVERS=%s", ic.KafkaBootstrap),
			// dev-container v1: PLAINTEXT listener (SCRAM bootstrap user reserved for a follow-on).
			"KAFKA_SECURITY_PROTOCOL=PLAINTEXT",
		}
		if ic.SchemaRegistryHTTP != "" {
			lines = append(lines, fmt.Sprintf("SCHEMA_REGISTRY_URL=%s", ic.SchemaRegistryHTTP))
		}
		return strings.Join(lines, "\n")
	case "unimeter":
		if ic.UnimeterAPI == "" {
			return ""
		}
		lines := []string{fmt.Sprintf("UNIMETER_ADDR=%s", ic.UnimeterAPI)}
		if ic.UnimeterHealthHTTP != "" {
			lines = append(lines, fmt.Sprintf("UNIMETER_HEALTH_URL=%s", ic.UnimeterHealthHTTP))
		}
		return strings.Join(lines, "\n")
	case "browserless":
		if ic.BrowserlessHTTP == "" {
			return ""
		}
		lines := []string{
			fmt.Sprintf("BROWSERLESS_URL=%s", ic.BrowserlessHTTP),
			fmt.Sprintf("BROWSERLESS_WS_ENDPOINT=%s", ic.BrowserlessWS),
		}
		if tok := strings.TrimSpace(inst.Credentials.AccessKey); tok != "" {
			lines = append(lines, fmt.Sprintf("BROWSERLESS_TOKEN=%s", tok))
		}
		return strings.Join(lines, "\n")
	case "bunqueue":
		if ic.BunqueueTCP == "" {
			return ""
		}
		lines := []string{fmt.Sprintf("BUNQUEUE_ADDR=%s", ic.BunqueueTCP)}
		if ic.BunqueueHealthHTTP != "" {
			lines = append(lines, fmt.Sprintf("BUNQUEUE_HEALTH_URL=%s", ic.BunqueueHealthHTTP))
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

// schemaRegistryPublicURL is the Schema Registry base URL clients use from outside the cluster.
func schemaRegistryPublicURL(spec Spec, inst ServiceInstance, host string) string {
	if VPSEdge(spec) {
		if edge := PublicEdgeHost(inst); edge != "" && strings.TrimSpace(os.Getenv("OBELISK_DEV")) != "" {
			return fmt.Sprintf("http://%s:8081", edge)
		}
		if u := PublicHTTPEdgeURL(inst); u != "" {
			return u
		}
	}
	if h := strings.TrimSpace(host); h != "" {
		return fmt.Sprintf("http://%s:8081", h)
	}
	return ""
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

// edgeConnectionTLS describes TLS for native protocols behind the VPS SNI edge.
func edgeConnectionTLS(inst ServiceInstance) *ConnectionTLS {
	switch inst.Type {
	case "postgres":
		return &ConnectionTLS{Enabled: true, ServerName: PublicEdgeHost(inst), Negotiation: "direct"}
	case "mysql", "mongodb", "redis":
		return &ConnectionTLS{Enabled: true, ServerName: PublicEdgeHost(inst)}
	}
	return nil
}
