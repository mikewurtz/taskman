package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	testUserID     = "client001"
	testServerAddr = "localhost:50055"
	testTimeout    = 5 * time.Second
)

var stopServer func()

func TestMain(m *testing.M) {
	stopServer = startTestServer()

	code := m.Run()

	if stopServer != nil {
		stopServer()
	}
	os.Exit(code)
}

func startTestServer() func() {
	srv, err := server.NewServer(testServerAddr)
	if err != nil {
		panic(fmt.Sprintf("failed to start test server: %v", err))
	}

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("Test server error: %v\n", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	return func() {
		srv.Stop()
	}
}

func createTestClient(t *testing.T, userID string) pb.TaskManagerClient {
	t.Helper()

	if userID == "" {
		userID = testUserID
	}

	certPath := fmt.Sprintf("%s.crt", userID)
	keyPath := fmt.Sprintf("%s.key", userID)

	certPEM, err := certs.CertFiles.ReadFile(certPath)
	require.NoError(t, err, "failed to read embedded client cert")

	keyPEM, err := certs.CertFiles.ReadFile(keyPath)
	require.NoError(t, err, "failed to read embedded client key")

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err, "failed to parse key pair")

	caCert, err := certs.CertFiles.ReadFile("ca.crt")
	require.NoError(t, err, "failed to read embedded CA cert")

	caPool := x509.NewCertPool()
	require.True(t, caPool.AppendCertsFromPEM(caCert), "failed to append CA cert to pool")

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	conn, err := grpc.NewClient(
		testServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	)
	require.NoError(t, err, "failed to set up gRPC client")

	t.Cleanup(func() {
		err := conn.Close()
		require.NoError(t, err, "failed to clean up gRPC connectiont")
	})

	return pb.NewTaskManagerClient(conn)
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
