package integration

import (
	"context"
	"testing"

	pb "github.com/mikewurtz/taskman/gen/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func TestIntegration_GetTaskStatusDoesNotExist(t *testing.T) {
	t.Parallel()

	client := createTestClient(t, "client001")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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
