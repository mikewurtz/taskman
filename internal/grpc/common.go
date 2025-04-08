package grpc

import (
	"crypto/x509"
	"fmt"

	"github.com/mikewurtz/taskman/certs"
)

func LoadCACertPool() (*x509.CertPool, error) {
	caCert, err := certs.CertFiles.ReadFile("ca.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}
	return pool, nil
}
