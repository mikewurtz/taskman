package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/mikewurtz/taskman/certs"
)

// LoadCACertPool loads the CA certificate pool from the embedded files
func LoadCACertPool() (*x509.CertPool, error) {
	caCert, err := certs.CertFiles.ReadFile(certs.CACertFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, errors.New("failed to append CA certificate")
	}
	return pool, nil
}

// LoadTLSCert loads a TLS certificate from the embedded files
func LoadTLSCert(certName string) (tls.Certificate, error) {
	certPEM, err := certs.CertFiles.ReadFile(certName + ".crt")
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read cert file: %w", err)
	}

	keyPEM, err := certs.CertFiles.ReadFile(certName + ".key")
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to read key file: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse key pair: %w", err)
	}

	return cert, nil
}
