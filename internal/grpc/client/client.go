package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/mikewurtz/taskman/gen/proto"
	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
)

// NewClient creates a new gRPC client with mTLS authentication
func NewClient(userID string, serverAddr string) (pb.TaskManagerClient, *grpc.ClientConn, error) {
	cert, err := loadClientCert(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading client cert: %w", err)
	}

	caPool, err := loadCACert()
	if err != nil {
		return nil, nil, fmt.Errorf("loading CA cert: %w", err)
	}

	conn, err := createConnection(serverAddr, cert, caPool)
	if err != nil {
		return nil, nil, fmt.Errorf("creating connection: %w", err)
	}

	return pb.NewTaskManagerClient(conn), conn, nil
}

func loadClientCert(userID string) (tls.Certificate, error) {
	certFile := fmt.Sprintf("certs/%s.crt", userID)
	keyFile := fmt.Sprintf("certs/%s.key", userID)
	return tls.LoadX509KeyPair(certFile, keyFile)
}

func loadCACert() (*x509.CertPool, error) {
	caCert, err := os.ReadFile(basegrpc.CaCertPath)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}
	return pool, nil
}

func createConnection(addr string, cert tls.Certificate, caPool *x509.CertPool) (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	return grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}
