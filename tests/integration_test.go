package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
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
	testTimeout    = 5 * time.Second
	exitCode2      = int32(2)
	exitCodeKilled = int32(-1)
	testUserID = "client001"
)

var (
	stopServer     func()
	testServerAddr string
	once           sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()

	if stopServer != nil {
		stopServer()
	}
	os.Exit(code)
}

// startTestServer starts the test server and returns a function to stop it
// will only be called once
func startTestServer(t *testing.T) {
	once.Do(func() {
		fmt.Println("starting test server")
		srv, err := server.NewServer("localhost:0")
		require.NoError(t, err, "failed to create test server")

		go func() {
			if err := srv.Start(); err != nil {
				fmt.Printf("Test server error: %v\n", err)
			}
		}()

		testServerAddr = srv.Addr()
		// wait for the server to be ready to handle requests, 2 seconds should be plenty
		require.Eventually(t, func() bool {
			client := createTestClient(t, testUserID)
			// set context timeout to 500ms so we can still retry a few times if it fails
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			// try a get task status with a task that does not exist, if it returns codes.NotFound then the server is ready
			_, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
				TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
			})

			sts, ok := status.FromError(err)
			// check that the error is NotFound; we should be ready if it is
			return ok && sts.Code() == codes.NotFound
		}, 2*time.Second, 10*time.Millisecond, "gRPC server did not start in time")

		stopServer = func() {
			srv.Stop()
		}
	})
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
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"-l"},
	})
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)
	require.NoError(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.OK, sts.Code())
}

func TestIntegration_StartTaskWithFullPath(t *testing.T) {
	t.Parallel()
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "/bin/ls",
		Args:    []string{"-l"},
	})
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)
	require.NoError(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.OK, sts.Code())
}

func TestIntegration_StartTaskStopImmediately(t *testing.T) {
	t.Parallel()
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "sleep",
		Args:    []string{"5"},
	})
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)
	require.NoError(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.OK, sts.Code())

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	// stop the task immediately
	stopResp, err := client.StopTask(stopCtx, &pb.StopTaskRequest{
		TaskId: resp.TaskId,
	})
	require.NoError(t, err)
	stopSts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.OK, stopSts.Code())
	assert.NotNil(t, stopResp)

	// get the status of the task
	var statusResp *pb.TaskStatusResponse
	const pollInterval = 100 * time.Millisecond
	require.Eventually(t, func() bool {
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && statusResp != nil && statusResp.Status == pb.JobStatus_JOB_STATUS_SIGNALED
	}, 3*time.Second, pollInterval, "expected task to exit with error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_SIGNALED, statusResp.Status)
	assert.Equal(t, "user", statusResp.TerminationSource)
	assert.Equal(t, "killed", statusResp.TerminationSignal)
	assert.NotNil(t, statusResp.EndTime)
	assert.NotNil(t, statusResp.StartTime)
	assert.Equal(t, resp.TaskId, statusResp.TaskId)
	assert.Nil(t, statusResp.ExitCode)
}

func TestIntegration_StartTaskExitsError(t *testing.T) {
	t.Parallel()
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"/nonexistent"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TaskId)

	var statusResp *pb.TaskStatusResponse
	const pollInterval = 100 * time.Millisecond

	require.Eventually(t, func() bool {
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && statusResp != nil && statusResp.Status == pb.JobStatus_JOB_STATUS_EXITED_ERROR
	}, 3*time.Second, pollInterval, "expected task to exit with error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_EXITED_ERROR, statusResp.Status)
	assert.Equal(t, "", statusResp.TerminationSource)
	assert.Equal(t, "", statusResp.TerminationSignal)
	assert.NotNil(t, statusResp.EndTime)
	assert.NotNil(t, statusResp.StartTime)
	assert.Equal(t, resp.TaskId, statusResp.TaskId)
	assert.Equal(t, exitCode2, *statusResp.ExitCode)
}

func TestIntegration_StartTaskCommandDoesNotExistInPath(t *testing.T) {
	t.Parallel()
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "test-command-that-does-not-exist",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, sts.Code())
	assert.Contains(t, sts.Message(), "invalid command")
	assert.Contains(t, sts.Message(), "executable file not found in $PATH")
}

func TestIntegration_StartTaskFullPathCommandDoesNotExist(t *testing.T) {
	t.Parallel()
	startTestServer(t)

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "/path/to/test-command-that-does-not-exist",
	})
	require.Error(t, err)
	fmt.Println(err)
	require.Nil(t, resp)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, sts.Code())
	assert.Contains(t, sts.Message(), "command not found")
	assert.Contains(t, sts.Message(), "no such file or directory")
}

func TestIntegration_GetTaskStatusDoesNotExist(t *testing.T) {
	t.Parallel()
	startTestServer(t)

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
	assert.Equal(t, codes.NotFound, sts.Code())
}

func TestIntegration_StreamTaskOutput(t *testing.T) {
	t.Parallel()
	startTestServer(t)

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

func TestIntegration_StopTaskDoesNotExist(t *testing.T) {
	t.Parallel()
	startTestServer(t)

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
	assert.Equal(t, codes.NotFound, sts.Code())
}

func TestIntegration_StopTaskContextCanceled(t *testing.T) {
	t.Parallel()
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
	startTestServer(t)

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
