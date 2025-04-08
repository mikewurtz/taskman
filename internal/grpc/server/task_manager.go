package server

import (
	"context"

	pb "github.com/mikewurtz/taskman/gen/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

)

func NewTaskManagerServer() *taskManagerServer {
	return &taskManagerServer{}
}

type taskManagerServer struct {
	// this gives us a forward compatible implementation to extend later
	pb.UnimplementedTaskManagerServer
}

// TODO: Once we have a real implementation, we will break up this file into multiple files
// These functions will be moved to the appropriate files and utilize the reuseable library
// for managing the linux processes.

// StartTask
func (s *taskManagerServer) StartTask(ctx context.Context, req *pb.StartTaskRequest) (*pb.StartTaskResponse, error) {
	return nil, status.Error(codes.Unimplemented, "StartTask not implemented")
}

// StopTask
func (s *taskManagerServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "StopTask not implemented")
}

// GetTaskStatus
func (s *taskManagerServer) GetTaskStatus(ctx context.Context, req *pb.TaskStatusRequest) (*pb.TaskStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "GetTaskStatus not implemented")
}

// StreamTaskOutput
func (s *taskManagerServer) StreamTaskOutput(req *pb.StreamTaskOutputRequest, stream pb.TaskManager_StreamTaskOutputServer) error {
	return status.Errorf(codes.Unimplemented, "StreamTaskOutput not implemented")
}
