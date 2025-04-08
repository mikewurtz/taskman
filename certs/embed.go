package certs

import (
	"embed"
)

// CertFiles holds embedded certificates and keys.
// These are included directly in the binary for simplicity in this demo.
// In a production environment, keys should be loaded from a secure store,
// rotated regularly, and not checked into version control.

//go:embed *.crt *.key
var CertFiles embed.FS
