package grpc

import (
	"crypto/x509"
	"fmt"
	"os"
)

func LoadCACertPool() (*x509.CertPool, error) {
	caCert, err := os.ReadFile(CaCertPath)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}
	return pool, nil
}
