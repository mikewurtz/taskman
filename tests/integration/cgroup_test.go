package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	pb "github.com/mikewurtz/taskman/gen/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseCPUStat(t *testing.T, content string) map[string]uint64 {
	t.Helper()
	stats := make(map[string]uint64)
	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("invalid cpu.stat line: %q", line)
		}
		val, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			t.Fatalf("invalid value in cpu.stat line: %q: %v", line, err)
		}
		stats[parts[0]] = val
	}
	return stats
}

func TestIntegration_StartTaskOOMKilledPerl(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "perl",
		Args:    []string{"-e", "my $x = \"A\" x (128 * 1024 * 1024); sleep 5;"},
	})
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)
	require.NoError(t, err)
	sts, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.OK, sts.Code())

	var statusResp *pb.TaskStatusResponse
	require.Eventually(t, func() bool {
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		// wait for the task to exit with and error
		return err == nil && statusResp != nil && statusResp.Status == pb.JobStatus_JOB_STATUS_SIGNALED
	}, 3*time.Second, pollInterval, "expected task to exit with error")

	assert.Equal(t, pb.JobStatus_JOB_STATUS_SIGNALED, statusResp.Status)
	assert.Equal(t, "oom", statusResp.TerminationSource)
	assert.Equal(t, syscall.SIGKILL.String(), statusResp.TerminationSignal)
	assert.NotEmpty(t, statusResp.ProcessId)
	assert.NotNil(t, statusResp.EndTime)
	assert.NotNil(t, statusResp.StartTime)
	assert.Equal(t, resp.TaskId, statusResp.TaskId)
	assert.Nil(t, statusResp.ExitCode)
}

func TestIntegration_StartTaskIOThrottled(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "dd",
		Args: []string{
			"if=/dev/sda",
			"of=/dev/null",
			"bs=1M",
			"count=5",
			"iflag=direct",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)

	require.Eventually(t, func() bool {

		ioStatPath := filepath.Join("/sys/fs/cgroup", resp.TaskId, "io.stat")
		data, readErr := os.ReadFile(ioStatPath)
		if readErr != nil {
			t.Logf("waiting for io.stat: %v", readErr)
			return false
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) == 0 {
			return false
		}

		fields := strings.Fields(lines[0])
		if len(fields) < 2 {
			return false
		}

		stats := make(map[string]uint64)
		for _, field := range fields[1:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) == 2 {
				val, err := strconv.ParseUint(parts[1], 10, 64)
				if err == nil {
					stats[parts[0]] = val
				}
			}
		}

		return stats["rbytes"] > 0

	}, 10*time.Second, 100*time.Millisecond, "expected task to complete and io.stat to reflect I/O")

	// stop the task to clean up after ourselves
	// the stop request will return ok if the task is stopped with our request
	// or failed precondition if the task if already completed
	_, err = client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: resp.TaskId,
	})
	sts, ok := status.FromError(err)
	require.True(t, ok)
	require.True(t, sts.Code() == codes.OK || sts.Code() == codes.FailedPrecondition)
}

func TestIntegration_CPUThrottled_BashLoop(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Start a CPU-bound task using bash busy loop
	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
		Command: "bash",
		Args:    []string{"-c", "while :; do :; done"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.TaskId)

	var (
		statusResp *pb.TaskStatusResponse
		cpuStats   map[string]uint64
	)

	require.Eventually(t, func() bool {
		// Confirm the task is running
		statusResp, err = client.GetTaskStatus(ctx, &pb.TaskStatusRequest{
			TaskId: resp.TaskId,
		})
		if err != nil || statusResp == nil || statusResp.Status != pb.JobStatus_JOB_STATUS_STARTED {
			return false
		}

		// Try reading cpu.stat while task is running
		cpuStatPath := filepath.Join("/sys/fs/cgroup", resp.TaskId, "cpu.stat")
		data, readErr := os.ReadFile(cpuStatPath)
		if readErr != nil {
			t.Logf("cpu.stat not ready yet: %v", readErr)
			return false
		}
		cpuStats = parseCPUStat(t, string(data))
		return cpuStats["nr_throttled"] > 0 && cpuStats["throttled_usec"] > 0
	}, 10*time.Second, pollInterval, "expected task to start and cpu.stat to show throttling")

	// stop the task to clean up after ourselves
	// the stop request will return ok if the task is stopped with our request
	// or failed precondition if the task if already completed
	_, err = client.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: resp.TaskId,
	})
	sts, ok := status.FromError(err)
	require.True(t, ok)
	require.True(t, sts.Code() == codes.OK || sts.Code() == codes.FailedPrecondition)
}
