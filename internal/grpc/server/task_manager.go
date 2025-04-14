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

// taskManagerServer is the implementation of the TaskManager service
type taskManagerServer struct {
	// this gives us a forward compatible implementation to extend later
	pb.UnimplementedTaskManagerServer
	taskManager *taskmanager.TaskManager
}

// StartTask starts a new task and returns the task ID
func (s *taskManagerServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	taskID, err := s.taskManager.StartTask(ctx, req.Command, req.Args)
	if err != nil {
		return nil, task.TaskErrorToGRPC(err)
	}
	return &pb.StartTaskResponse{TaskId: taskID}, nil
}

// StopTask stops the task with the given ID
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

// GetTaskStatus returns the status of the task with the given ID
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

	snapshot := taskObj.Snapshot()
	returnStatus := &pb.TaskStatusResponse{
		TaskId:            snapshot.ID,
		ProcessId:         int32(snapshot.ProcessID),
		Status:            status,
		StartTime:         timestamppb.New(snapshot.StartTime),
		EndTime:           timestamppb.New(snapshot.EndTime),
		ExitCode:          snapshot.ExitCode,
		TerminationSignal: snapshot.TerminationSignal,
		TerminationSource: snapshot.TerminationSource,
	}

	return returnStatus, nil
}

func checkAuthorization(caller string, taskObj *taskmanager.Task) error {
	if taskObj.GetClientID() != caller && caller != "admin" {
		return status.Errorf(codes.NotFound, "task with id %s not found", taskObj.GetID())
	}
	return nil
}

// StreamTaskOutput outputs the task’s streams to the client by tracking its read offset.
func (s *taskManagerServer) StreamTaskOutput(req *pb.StreamTaskOutputRequest, stream pb.TaskManager_StreamTaskOutputServer) error {
	taskObj, err := s.taskManager.GetTask(stream.Context(), req.TaskId)
	if err != nil {
		return task.TaskErrorToGRPC(err)
	}

	caller := stream.Context().Value(basegrpc.ClientIDKey).(string)
	if err = checkAuthorization(caller, taskObj); err != nil {
		return err
	}

	// Create a writer closure that sends data over the gRPC stream
	// stream.Send() will block if the client is slow to read the data
	writer := func(data []byte) error {
		return stream.Send(&pb.StreamTaskOutputResponse{Output: data})
	}

	err = s.taskManager.StreamTaskOutput(stream.Context(), req.TaskId, writer)
	if err != nil {
		return task.TaskErrorToGRPC(err)
	}

	return nil
}
