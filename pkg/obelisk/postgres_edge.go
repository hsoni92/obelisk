package obelisk

// VPSPostgresURLQuery is required on postgresql:// URLs published for the native TLS edge.
// Nginx stream routes on the first TLS ClientHello (SNI); classic libpq SSLRequest is not TLS and gets reset.
const VPSPostgresURLQuery = "sslmode=require&sslnegotiation=direct"