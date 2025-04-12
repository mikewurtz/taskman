package server

import (
	"context"

	pb "github.com/mikewurtz/taskman/gen/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
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
	taskObj, err := s.taskManager.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if err = checkAuthorization(caller, taskObj); err != nil {
		return nil, err
	}
	if err := s.taskManager.StopTask(ctx, req.TaskId); err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	return &pb.StopTaskResponse{}, nil
}

// GetTaskStatus
func (s *taskManagerServer) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	taskObj, err := s.taskManager.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if err = checkAuthorization(caller, taskObj); err != nil {
		return nil, err
	}
	status, err := task.StatusToProto(taskObj.GetStatus())
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}

	returnStatus := &pb.TaskStatusResponse{
		TaskId:            taskObj.GetID(),
		ProcessId:         int32(taskObj.GetProcessID()),
		Status:            status,
		StartTime:         timestamppb.New(taskObj.GetStartTime()),
		EndTime:           timestamppb.New(taskObj.GetEndTime()),
		ExitCode:          taskObj.GetExitCode(),
		TerminationSignal: taskObj.GetTerminationSignal(),
		TerminationSource: taskObj.GetTerminationSource(),
	}

	return returnStatus, nil
}

func checkAuthorization(caller string, taskObj *taskmanager.Task) error {
	if taskObj.GetClientID() != caller && caller != "admin" {
		return status.Errorf(codes.NotFound, "task with id %s not found", taskObj.GetID())
	}
	return nil
}

// StreamTaskOutput
func (s *taskManagerServer) StreamTaskOutput(req *pb.StreamTaskOutputRequest, stream pb.TaskManager_StreamTaskOutputServer) error {
	return status.Errorf(codes.Unimplemented, "StreamTaskOutput not implemented")
}
