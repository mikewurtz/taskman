package server

import (
	"context"
	"errors"

	pb "github.com/mikewurtz/taskman/gen/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mikewurtz/taskman/internal/task"
	taskmanager "github.com/mikewurtz/taskman/internal/task/manager"
)

func NewTaskManagerServer() *taskManagerServer {
	return &taskManagerServer{
		taskManager: taskmanager.NewTaskManager(),
	}
}

type taskManagerServer struct {
	// this gives us a forward compatible implementation to extend later
	pb.UnimplementedTaskManagerServer
	taskManager *taskmanager.TaskManager
}

// handleTaskError converts a TaskError to a gRPC status error
func handleTaskError(err error) error {
	var taskErr *task.TaskError
	if errors.As(err, &taskErr) {
		return status.Error(taskErr.Code, taskErr.Error())
	}
	// If it's not a TaskError, return it as an internal error
	return status.Error(codes.Internal, err.Error())
}

// StartTask
func (s *taskManagerServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	taskID, err := s.taskManager.StartTask(ctx, req.Command, req.Args)
	if err != nil {
		return nil, handleTaskError(err)
	}
	return &pb.StartTaskResponse{TaskId: taskID}, nil
}

// StopTask
func (s *taskManagerServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	if err := s.taskManager.StopTask(ctx, req.TaskId); err != nil {
		return nil, handleTaskError(err)
	}
	return &pb.StopTaskResponse{}, nil
}

// GetTaskStatus
func (s *taskManagerServer) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	task, err := s.taskManager.GetTaskStatus(ctx, req.TaskId)
	if err != nil {
		return nil, handleTaskError(err)
	}

	returnStatus := &pb.TaskStatusResponse{
		TaskId:            task.ID,
		ProcessId:         int32(task.ProcessID),
		Status:            pb.JobStatus(pb.JobStatus_value[task.Status]),
		StartTime:         timestamppb.New(task.StartTime),
		EndTime:           timestamppb.New(task.EndTime),
		ExitCode:          task.ExitCode,
		TerminationSignal: task.TerminationSignal,
		TerminationSource: task.TerminationSource,
	}

	return returnStatus, nil
}

// StreamTaskOutput
func (s *taskManagerServer) StreamTaskOutput(req *pb.StreamTaskOutputRequest, stream pb.TaskManager_StreamTaskOutputServer) error {
	stream.Send(&pb.StreamTaskOutputResponse{})
	return status.Errorf(codes.Unimplemented, "StreamTaskOutput not implemented")
}
