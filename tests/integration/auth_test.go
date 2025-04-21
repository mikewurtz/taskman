package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/mikewurtz/taskman/gen/proto"
)

// tests a valid client cert but has no common name for unary calls (stop, start, get-status)
func TestIntegration_NoCNInKeyUnary(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client-no-cn")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, sts.Code())
}

// tests a valid client cert but has no common name for stream call
func TestIntegration_NoCNInKeyStream(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client-no-cn")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.NoError(t, err)

	resp, err := stream.Recv()
	require.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, sts.Code())

}

// tests a key that is self signed and not by a CA
func TestIntegration_SelfSignedCertNoCA(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "badclient-self-signed")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, sts.Code())
}

// tests a weak key that is RSA 512
func TestIntegration_WeakKey512(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "weak")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "375b0522-72ed-4f3f-88d0-01d360d06b8c",
	})
	require.Nil(t, resp)
	require.Error(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unavailable, sts.Code())
}

func TestIntegration_StartTaskTestAuthorization(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
	})
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TaskId)
	require.NoError(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.OK, sts.Code())

	// try to get the task status using a different client
	client2 := createTestClient(t, "client002")
	statusResp, err := client2.GetTaskStatus(ctx, &pb.TaskStatusRequest{
		TaskId: resp.TaskId,
	})
	require.Nil(t, statusResp)
	require.Error(t, err)
	sts, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, sts.Code())

	// try to stop the task status using a different client
	_, err = client2.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: resp.TaskId,
	})
	require.Error(t, err)
	sts, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, sts.Code())

	// Try to stream the task output using a different client (unauthorized)
	stream, err := client2.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{
		TaskId: resp.TaskId,
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)

	sts, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, sts.Code())

	// get the status of the task using the admin client
	adminClient := createTestClient(t, "admin")
	var adminStatusResp *pb.TaskStatusResponse
	require.Eventually(t, func() bool {
		adminStatusResp, err = adminClient.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && adminStatusResp != nil && adminStatusResp.Status == pb.JobStatus_JOB_STATUS_EXITED_OK
	}, 3*time.Second, pollInterval, "expected task to exit without error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_EXITED_OK, adminStatusResp.Status)
	assert.Empty(t, adminStatusResp.TerminationSource)
	assert.Empty(t, adminStatusResp.TerminationSignal)
	assert.NotNil(t, adminStatusResp.EndTime)
	assert.NotNil(t, adminStatusResp.StartTime)
	assert.NotEmpty(t, adminStatusResp.ProcessId)
	assert.Equal(t, resp.TaskId, adminStatusResp.TaskId)
	assert.NotNil(t, adminStatusResp.ExitCode)
	assert.Equal(t, int32(0), *adminStatusResp.ExitCode)

}
