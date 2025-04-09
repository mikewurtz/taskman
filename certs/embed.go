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

// In a real environment these would be set via environment variables and not be hardcoded
// we would want to use a config package to manage these values and could load these in from env vars and utilize
// https://github.com/kelseyhightower/envconfig

const (
	ServerCertName = "server"
	CACertFileName = "ca.crt"
)
