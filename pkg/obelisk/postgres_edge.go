package obelisk

// VPSPostgresURLQuery is required on postgresql:// URLs published for the native TLS edge.
// Nginx stream routes on the first TLS ClientHello (SNI); classic libpq SSLRequest is not TLS and gets reset.
// With sslnegotiation=direct the client must offer TLS ALPN "postgresql" (libpq and pgx do this automatically).
const VPSPostgresURLQuery = "sslmode=require&sslnegotiation=direct"