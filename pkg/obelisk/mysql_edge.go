package obelisk

// VPSMySQLURLQuery is appended to mysql:// URLs on the native TLS edge (port 9443).
// Clients must open TLS before the MySQL handshake (nginx SNI passthrough); see admin connection notes.
const VPSMySQLURLQuery = "tls=true"
