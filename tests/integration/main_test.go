package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mikewurtz/taskman/certs"
	"github.com/mikewurtz/taskman/internal/grpc/server"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	pb "github.com/mikewurtz/taskman/gen/proto"
)

const (
	testTimeout  = 5 * time.Second
	pollInterval = 100 * time.Millisecond
	exitCode2    = int32(2)
	testUserID   = "client001"
)

var (
	testServerAddr string
)

func TestMain(m *testing.M) {
	stopServer, err := startTestServer()
	if err != nil {
		fmt.Println("failed to start test server:", err)
		os.Exit(1)
	}

	code := m.Run()
	if stopServer != nil {
		stopServer()
	}
	os.Exit(code)
}

// startTestServer starts the test server and returns a function to stop it
// will only be called once
func startTestServer() (func(), error) {
	srv, err := server.New("localhost:0", context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create test server: %w", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("Test server error: %v\n", err)
		}
	}()

	stopServer := func() {
		srv.Stop()
	}

	testServerAddr = srv.Addr()
	client, conn, err := createClient(testUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("failed to close gRPC connection: %v\n", err)
		}
	}()

	// wait for the server to be ready to handle requests, 2 seconds should be plenty
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		// set context timeout to 500ms so we can still retry a few times if it fails
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
		})
		cancel()

		sts, ok := status.FromError(err)
		if ok && sts.Code() == codes.NotFound {
			// Server is ready we can return
			return stopServer, nil
		}
		// Pause briefly before retrying
		time.Sleep(10 * time.Millisecond)
	}

	return nil, errors.New("gRPC server did not start in time")
}

func createClient(userID string) (pb.TaskManagerClient, *grpc.ClientConn, error) {
	if userID == "" {
		userID = testUserID
	}

	certPath := userID + ".crt"
	keyPath := userID + ".key"

	certPEM, err := certs.CertFiles.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read embedded client cert: %w", err)
	}

	keyPEM, err := certs.CertFiles.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read embedded client key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse key pair: %w", err)
	}

	caCert, err := certs.CertFiles.ReadFile("ca.crt")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read embedded CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, nil, errors.New("failed to append CA cert to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	conn, err := grpc.NewClient(
		testServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set up gRPC client: %w", err)
	}

	return pb.NewTaskManagerClient(conn), conn, nil
}

func createTestClient(t *testing.T, userID string) pb.TaskManagerClient {
	t.Helper()

	client, conn, err := createClient(userID)
	require.NoError(t, err, "failed to create gRPC client")

	t.Cleanup(func() {
		err := conn.Close()
		require.NoError(t, err, "failed to clean up gRPC connection")
	})
	return client
}
