package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/mikewurtz/taskman/gen/proto"
)

type mockStream struct {
	pb.TaskManager_StreamTaskOutputServer
	ctx context.Context
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func TestNewTaskManagerServer(t *testing.T) {
	t.Parallel()
	server := NewTaskManagerServer()
	assert.NotNil(t, server)
}

func TestStartTask_Unimplemented(t *testing.T) {
	t.Parallel()
	server := NewTaskManagerServer()
	ctx := context.Background()

	resp, err := server.StartTask(ctx, &pb.StartTaskRequest{
		Command: "echo",
		Args:    []string{"hello"},
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	assert.Contains(t, err.Error(), "StartTask not implemented")
}

func TestStopTask_Unimplemented(t *testing.T) {
	t.Parallel()
	server := NewTaskManagerServer()
	ctx := context.Background()

	resp, err := server.StopTask(ctx, &pb.StopTaskRequest{
		TaskId: "test-task",
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	assert.Contains(t, err.Error(), "StopTask not implemented")
}

func TestGetTaskStatus_Unimplemented(t *testing.T) {
	t.Parallel()
	server := NewTaskManagerServer()
	ctx := context.Background()
	resp, err := server.GetTaskStatus(ctx, &pb.TaskStatusRequest{
		TaskId: "test-task",
	})

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	assert.Contains(t, err.Error(), "GetTaskStatus not implemented")
}

func TestStreamTaskOutput_Unimplemented(t *testing.T) {
	t.Parallel()
	server := NewTaskManagerServer()
	ctx := context.Background()

	stream := &mockStream{ctx: ctx}
	err := server.StreamTaskOutput(&pb.StreamTaskOutputRequest{
		TaskId: "test-task",
	}, stream)

	assert.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
	assert.Contains(t, err.Error(), "StreamTaskOutput not implemented")
}