package integration

import (
	"context"
	"syscall"
	"testing"
	"time"

	pb "github.com/mikewurtz/taskman/gen/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func TestIntegration_StartTaskExitsError(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "ls",
		Args:    []string{"/nonexistent"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.TaskId)

	var statusResp *pb.TaskStatusResponse

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

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "/path/to/test-command-that-does-not-exist",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, sts.Code())
	assert.Contains(t, sts.Message(), "command not found")
	assert.Contains(t, sts.Message(), "no such file or directory")
}

func TestIntegration_StartTask(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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
	require.Eventually(t, func() bool {
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && statusResp != nil && statusResp.Status == pb.JobStatus_JOB_STATUS_SIGNALED
	}, 3*time.Second, pollInterval, "expected task to exit with error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_SIGNALED, statusResp.Status)
	assert.Equal(t, "user", statusResp.TerminationSource)
	assert.Equal(t, syscall.SIGKILL.String(), statusResp.TerminationSignal)
	assert.NotNil(t, statusResp.EndTime)
	assert.NotNil(t, statusResp.StartTime)
	assert.NotEmpty(t, statusResp.ProcessId)
	assert.Equal(t, resp.TaskId, statusResp.TaskId)
	assert.Nil(t, statusResp.ExitCode)
}

func TestIntegration_StartTaskStopImmediatelyAttemptToStopAgain(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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
	require.Eventually(t, func() bool {
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && statusResp != nil && statusResp.Status == pb.JobStatus_JOB_STATUS_SIGNALED
	}, 3*time.Second, pollInterval, "expected task to exit with error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_SIGNALED, statusResp.Status)
	assert.Equal(t, "user", statusResp.TerminationSource)
	assert.Equal(t, syscall.SIGKILL.String(), statusResp.TerminationSignal)
	assert.NotNil(t, statusResp.EndTime)
	assert.NotNil(t, statusResp.StartTime)
	assert.NotEmpty(t, statusResp.ProcessId)
	assert.Equal(t, resp.TaskId, statusResp.TaskId)
	assert.Nil(t, statusResp.ExitCode)

	// attempt to stop the task again
	stopResp2, err := client.StopTask(stopCtx, &pb.StopTaskRequest{
		TaskId: resp.TaskId,
	})
	require.Error(t, err)
	stopSts2, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, stopSts2.Code())
	assert.Nil(t, stopResp2)
	assert.Contains(t, stopSts2.Message(), "task has already completed")
}
