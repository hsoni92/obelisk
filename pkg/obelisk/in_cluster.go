package obelisk

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ConnectionAudience selects which copy-ready URLs to surface by default.
const (
	ConnectionAudienceEdge      = "edge"
	ConnectionAudienceInCluster = "in_cluster"
)

// ConnectionTLS tells a client exactly how to handle TLS for an instance so it needs no
// per-service special-casing. Obelisk edge certificates are self-signed, so Verify is false:
// clients must encrypt but not validate the chain (libpq sslmode=require semantics).
// When the URL already carries the required TLS options this is informational only.
type ConnectionTLS struct {
	Enabled bool `json:"enabled"`
	// Verify is false when the certificate is self-signed and must not be chain-validated.
	Verify bool `json:"verify"`
	// ServerName is the SNI name to present (edge routing is SNI-based).
	ServerName string `json:"server_name,omitempty"`
	// Negotiation is "direct" when TLS must start at the first byte (Postgres 17 sslnegotiation=direct).
	Negotiation string `json:"negotiation,omitempty"`
}

// ConnectionBlock is copy-ready URLs for one reachability audience.
type ConnectionBlock struct {
	TLS                *ConnectionTLS       `json:"tls,omitempty"`
	Use                string               `json:"use,omitempty"`
	Host               string               `json:"host,omitempty"`
	Postgres           string               `json:"postgres,omitempty"`
	MySQL              string               `json:"mysql,omitempty"`
	MongoDB            string               `json:"mongodb,omitempty"`
	Redis              string               `json:"redis,omitempty"`
	S3HTTP             string               `json:"s3_http,omitempty"`
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
}

// ObeliskNamespace is the Kubernetes namespace Obelisk workloads run in.
func ObeliskNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("OBELISK_NAMESPACE")); ns != "" {
		return ns
	}
	return "obelisk"
}

// ClusterServiceFQDN is the in-cluster DNS name for an instance's Kubernetes Service.
func ClusterServiceFQDN(inst ServiceInstance, namespace string) string {
	if namespace == "" {
		namespace = ObeliskNamespace()
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", inst.ReleaseName(), namespace)
}

// KongInClusterBaseURL is the HTTP base URL for Kong routes from pods in the cluster.
func KongInClusterBaseURL(namespace string) string {
	if namespace == "" {
		namespace = ObeliskNamespace()
	}
	return fmt.Sprintf("http://obelisk-kong.%s.svc.cluster.local:%s", namespace, AdminHTTPPortString())
}

// InClusterConnectionNotes documents in-cluster URLs for the admin UI and discovery.
func InClusterConnectionNotes(namespace string) string {
	if namespace == "" {
		namespace = ObeliskNamespace()
	}
	return "In-cluster: for apps running as pods on the same k3s/Kubernetes node. Use Kubernetes Service DNS in namespace " + namespace + " (works from any namespace). These URLs connect directly to chart listeners—no VPS TLS SNI edge on " + NativeEdgeTLSPort() + "."
}

func buildInClusterBlock(spec Spec, inst ServiceInstance, namespace string) ConnectionBlock {
	host := ClusterServiceFQDN(inst, namespace)
	var b ConnectionBlock
	b.Host = host

	user := inst.Credentials.Username
	if user == "" {
		user = "obelisk"
	}
	pass := url.QueryEscape(inst.Credentials.Password)
	if pass == "" {
		pass = "SECRET"
	}
	db := BootstrapDatabase(inst)
	// With edge TLS enabled the Mongo and Redis charts only listen with TLS, so the
	// in-cluster URLs must say so; otherwise clients connect in plaintext and get reset.
	tlsOnly := VPSEdge(spec) && (inst.Type == "mongodb" || inst.Type == "redis")
	if tlsOnly {
		b.TLS = &ConnectionTLS{Enabled: true, Verify: false, ServerName: host}
	} else if nativeTLSType(inst.Type) {
		// Explicitly plaintext, so clients never have to guess.
		b.TLS = &ConnectionTLS{}
	}

	switch inst.Type {
	case "postgres", "mysql", "mongodb":
		switch inst.Type {
		case "postgres":
			b.Postgres = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", user, pass, host, db)
		case "mysql":
			b.MySQL = fmt.Sprintf("mysql://%s:%s@tcp(%s:3306)/%s", user, pass, host, db)
		case "mongodb":
			b.MongoDB = fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?%s", user, pass, host, db, inClusterMongoQuery(tlsOnly))
		}
	case "redis":
		scheme := "redis"
		if tlsOnly {
			scheme = "rediss"
		}
		if inst.Credentials.Password != "" {
			b.Redis = fmt.Sprintf("%s://:%s@%s:6379/0", scheme, url.QueryEscape(inst.Credentials.Password), host)
		} else {
			b.Redis = fmt.Sprintf("%s://%s:6379/0", scheme, host)
		}
	case "rustfs":
		b.S3HTTP = fmt.Sprintf("http://%s:9000", host)
	case "qdrant":
		b.QdrantHTTP = fmt.Sprintf("http://%s:6333", host)
		b.QdrantGRPC = fmt.Sprintf("%s:6334", host)
	case "searxng":
		b.SearXNGHTTP = fmt.Sprintf("http://%s:8080", host)
	case "redpanda":
		b.KafkaBootstrap = fmt.Sprintf("%s:9092", host)
		b.SchemaRegistryHTTP = fmt.Sprintf("http://%s:8081", host)
	case "unimeter":
		b.UnimeterAPI = fmt.Sprintf("%s:7001", host)
		b.UnimeterHealthHTTP = fmt.Sprintf("http://%s:9090/health", host)
	case "bunqueue":
		b.BunqueueTCP = fmt.Sprintf("%s:6789", host)
		b.BunqueueHealthHTTP = fmt.Sprintf("http://%s:6790/health", host)
	case "browserless":
		b.BrowserlessHTTP = fmt.Sprintf("http://%s:3000", host)
		b.BrowserlessWS = fmt.Sprintf("ws://%s:3000", host)
	}

	if IsRelationalEngine(inst.Type) {
		for _, name := range DesiredDatabaseNames(inst) {
			if u := relationalConnectionURLInCluster(inst, name, host, tlsOnly); u != "" {
				b.Databases = append(b.Databases, DatabaseConnection{Name: name, URL: u})
			}
		}
	}

	ic := connectionBlockAsInstance(b)
	b.Use = instanceUse(spec, inst, ic)
	return b
}

func relationalConnectionURLInCluster(inst ServiceInstance, dbName, host string, tlsOnly bool) string {
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
	switch inst.Type {
	case "postgres":
		return fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable", user, pass, host, dbName)
	case "mysql":
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:3306)/%s", user, pass, host, dbName)
	case "mongodb":
		return fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?%s", user, pass, host, dbName, inClusterMongoQuery(tlsOnly))
	default:
		return ""
	}
}

func nativeTLSType(t string) bool {
	switch t {
	case "postgres", "mysql", "mongodb", "redis":
		return true
	}
	return false
}

// inClusterMongoQuery returns the query string for in-cluster Mongo URLs. When the chart enforces
// TLS (edge TLS on) the self-signed certificate must be accepted, which every driver supports via
// tlsAllowInvalidCertificates, so clients need no custom TLS code.
func inClusterMongoQuery(tlsOnly bool) string {
	if tlsOnly {
		return "authSource=admin&tls=true&tlsAllowInvalidCertificates=true"
	}
	return "authSource=admin"
}

func connectionBlockAsInstance(b ConnectionBlock) InstanceConnection {
	return InstanceConnection{
		Postgres:           b.Postgres,
		MySQL:              b.MySQL,
		MongoDB:            b.MongoDB,
		Redis:              b.Redis,
		S3HTTP:             b.S3HTTP,
		QdrantHTTP:         b.QdrantHTTP,
		QdrantGRPC:         b.QdrantGRPC,
		SearXNGHTTP:        b.SearXNGHTTP,
		BrowserlessHTTP:    b.BrowserlessHTTP,
		BrowserlessWS:      b.BrowserlessWS,
		UnimeterAPI:        b.UnimeterAPI,
		UnimeterHealthHTTP: b.UnimeterHealthHTTP,
		KafkaBootstrap:     b.KafkaBootstrap,
		SchemaRegistryHTTP: b.SchemaRegistryHTTP,
		Databases:          b.Databases,
	}
}

func attachInClusterBlock(ic *InstanceConnection, spec Spec, inst ServiceInstance, namespace string) {
	if ic == nil || !inst.Enabled {
		return
	}
	block := buildInClusterBlock(spec, inst, namespace)
	ic.InCluster = &block
}

// InstanceConnectionForAudience returns ic with edge (default) or in-cluster URLs at the top level.
func InstanceConnectionForAudience(ic InstanceConnection, audience string) InstanceConnection {
	audience = strings.ToLower(strings.TrimSpace(audience))
	if audience != ConnectionAudienceInCluster || ic.InCluster == nil {
		return ic
	}
	return applyConnectionBlock(ic, *ic.InCluster)
}

func applyConnectionBlock(ic InstanceConnection, b ConnectionBlock) InstanceConnection {
	out := ic
	out.Use = b.Use
	out.TLS = b.TLS
	out.Postgres = b.Postgres
	out.MySQL = b.MySQL
	out.MongoDB = b.MongoDB
	out.Redis = b.Redis
	out.S3HTTP = b.S3HTTP
	out.QdrantHTTP = b.QdrantHTTP
	out.QdrantGRPC = b.QdrantGRPC
	out.SearXNGHTTP = b.SearXNGHTTP
	out.BrowserlessHTTP = b.BrowserlessHTTP
	out.BrowserlessWS = b.BrowserlessWS
	out.UnimeterAPI = b.UnimeterAPI
	out.UnimeterHealthHTTP = b.UnimeterHealthHTTP
	out.KafkaBootstrap = b.KafkaBootstrap
	out.SchemaRegistryHTTP = b.SchemaRegistryHTTP
	out.Databases = b.Databases
	if b.Host != "" {
		out.PublicPath = b.Host
	}
	return out
}

func exposureInClusterBlock(ref, token, namespace string) *ConnectionBlock {
	var path string
	switch ref {
	case ExposureOpenRouterMember:
		path = "/openrouter/api/v1"
	case ExposureMailMember:
		path = "/mail/api/v1"
	default:
		return nil
	}
	base := KongInClusterBaseURL(namespace)
	url := base + path
	var b ConnectionBlock
	b.Host = strings.TrimPrefix(base, "http://")
	if token == "" {
		switch ref {
		case ExposureOpenRouterMember:
			b.Use = fmt.Sprintf("# OpenRouter gateway — token not configured on host\n%s", url)
		case ExposureMailMember:
			b.Use = fmt.Sprintf("# Mail gateway — token not configured on host\n%s", url)
		}
	} else {
		switch ref {
		case ExposureOpenRouterMember:
			b.Use = strings.Join([]string{
				"# OpenRouter gateway in-cluster (Bearer token)",
				"OPENROUTER_GATEWAY_URL=" + url,
				"OPENROUTER_GATEWAY_TOKEN=" + token,
				"Authorization: Bearer " + token,
			}, "\n")
		case ExposureMailMember:
			b.Use = strings.Join([]string{
				"# Mail gateway in-cluster (Bearer token)",
				"MAIL_GATEWAY_URL=" + url,
				"MAIL_GATEWAY_TOKEN=" + token,
				"Authorization: Bearer " + token,
			}, "\n")
		}
	}
	return &b
}
