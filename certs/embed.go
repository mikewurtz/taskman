package certs

import (
	"embed"
)

//go:embed *.crt *.key
var CertFiles embed.FS
