package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/mikewurtz/taskman/certs"
	pb "github.com/mikewurtz/taskman/gen/proto"
	"github.com/mikewurtz/taskman/internal/grpc/server"
)

const (
	testUserID = "client001"
)

var (
	stopServer     func()
	testServerAddr string
)

func TestMain(m *testing.M) {
	if err := startTestServer(); err != nil {
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
func startTestServer() error {
	srv, err := server.New("localhost:0")
	if err != nil {
		return fmt.Errorf("failed to create test server: %w", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("Test server error: %v\n", err)
		}
	}()

	stopServer = func() {
		srv.Stop()
	}

	testServerAddr = srv.Addr()
	client, conn, err := createClient(testUserID)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer conn.Close()

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
		if ok && sts.Code() == codes.Unimplemented {
			// Server is ready we can return
			return nil
		}
		// Pause briefly before retrying
		time.Sleep(10 * time.Millisecond)
	}

	return errors.New("gRPC server did not start in time")
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

func TestIntegration_StartTask(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"-l"},
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, sts.Code())
}

func TestIntegration_GetTaskStatus(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, sts.Code())
}

func TestIntegration_StreamTaskOutput(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.NoError(t, err)

	resp, err := stream.Recv()
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, sts.Code())
}

func TestIntegration_StopTask(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, sts.Code())
}

func TestIntegration_StopTaskContextCanceled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

func TestIntegration_StartTaskContextCanceled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"-l"},
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

func TestIntegration_GetTaskStatusContextCanceled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

func TestIntegration_StreamTaskOutputContextCanceled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})

	assert.Nil(t, stream)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

func TestIntegration_GetTaskStatusContextTimeout(t *testing.T) {
	t.Parallel()
	client := createTestClient(t, "client001")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"-l"},
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, sts.Code())
}

// tests a valid client cert but has no common name for unary calls (stop, start, get-status)
func TestIntegration_NoCNInKeyUnary(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client-no-cn")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

// tests a valid client cert but has no common name for stream call
func TestIntegration_NoCNInKeyStream(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client-no-cn")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.NoError(t, err)

	resp, err := stream.Recv()
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, sts.Code())

}

// tests a key that is self signed and not by a CA
func TestIntegration_SelfSignedCertNoCA(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "badclient-self-signed")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, sts.Code())
}

// tests a weak key that is RSA 512
func TestIntegration_WeakKey512(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "weak")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, sts.Code())
}
