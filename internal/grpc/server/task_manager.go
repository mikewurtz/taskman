package server

import (
	"context"

	pb "github.com/mikewurtz/taskman/gen/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mikewurtz/taskman/internal/task"
	taskmanager "github.com/mikewurtz/taskman/internal/task/manager"
)

func NewTaskManagerServer(ctx context.Context) *taskManagerServer {
	return &taskManagerServer{
		taskManager: taskmanager.NewTaskManager(ctx),
	}
}

type taskManagerServer struct {
	// this gives us a forward compatible implementation to extend later
	pb.UnimplementedTaskManagerServer
	taskManager *taskmanager.TaskManager
}

// StartTask
func (s *taskManagerServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	taskID, err := s.taskManager.StartTask(ctx, req.Command, req.Args)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	return &pb.StartTaskResponse{TaskId: taskID}, nil
}

// StopTask
func (s *taskManagerServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	if err := s.taskManager.StopTask(ctx, req.TaskId); err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	return &pb.StopTaskResponse{}, nil
}

// GetTaskStatus
func (s *taskManagerServer) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	taskObj, err := s.taskManager.GetTaskStatus(ctx, req.TaskId)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	status, err := task.StatusToProto(taskObj.Status)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}

	returnStatus := &pb.TaskStatusResponse{
		TaskId:            taskObj.ID,
		ProcessId:         int32(taskObj.ProcessID),
		Status:            status,
		StartTime:         timestamppb.New(taskObj.StartTime),
		EndTime:           timestamppb.New(taskObj.EndTime),
		ExitCode:          taskObj.ExitCode,
		TerminationSignal: taskObj.TerminationSignal,
		TerminationSource: taskObj.TerminationSource,
	}

	return returnStatus, nil
}

// StreamTaskOutput
func (s *taskManagerServer) StreamTaskOutput(req *pb.StreamTaskOutputRequest, stream pb.TaskManager_StreamTaskOutputServer) error {
	return status.Errorf(codes.Unimplemented, "StreamTaskOutput not implemented")
}
