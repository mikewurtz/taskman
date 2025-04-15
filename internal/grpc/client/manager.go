package client

import (
	"context"
	"fmt"
	"io"
	"os"

	"google.golang.org/grpc"

	pb "github.com/mikewurtz/taskman/gen/proto"
)

// Manager wraps the gRPC client operations
type Manager struct {
	client pb.TaskManagerClient
	conn   *grpc.ClientConn
}

// NewManager sets up a new gRPC manager
func NewManager(userID, serverAddr string) (*Manager, error) {
	client, conn, err := New(userID, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Manager{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (m *Manager) Close() error {
	return m.conn.Close()
}

// StartTask starts a new task with the given command and arguments
func (m *Manager) StartTask(ctx context.Context, command string, args []string) (string, error) {
	resp, err := m.client.StartTask(ctx, &pb.StartTaskRequest{
		Command: command,
		Args:    args,
	})
	if err != nil {
		return "", fmt.Errorf("error starting task: %w", err)
	}
	return resp.TaskId, nil
}

// GetTaskStatus gets the status of a task by its ID
func (m *Manager) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	pbStatus, err := m.client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: taskID})
	if err != nil {
		return nil, fmt.Errorf("error getting task status: %w", err)
	}

	returnStatus := &TaskStatus{
		TaskID:            pbStatus.TaskId,
		Status:            pbStatus.Status.String(),
		StartTime:         pbStatus.StartTime.AsTime(),
		EndTime:           pbStatus.EndTime.AsTime(),
		ExitCode:          pbStatus.ExitCode,
		ProcessID:         pbStatus.ProcessId,
		TerminationSignal: pbStatus.TerminationSignal,
		TerminationSource: pbStatus.TerminationSource,
	}

	return returnStatus, nil
}

// StreamTaskOutput streams the output of a task by its ID
func (m *Manager) StreamTaskOutput(ctx context.Context, taskID string) error {
	stream, err := m.client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error starting output stream: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// Stream completed normally
			return nil
		}
		if err != nil {
			return fmt.Errorf("error receiving stream: %w", err)
		}

		// Write raw bytes to stdout without UTF-8 conversion
		if _, err := os.Stdout.Write(resp.Output); err != nil {
			return fmt.Errorf("error writing to stdout: %w", err)
		}
	}
}

// StopTask stops a task by its ID
func (m *Manager) StopTask(ctx context.Context, taskID string) error {
	_, err := m.client.StopTask(ctx, &pb.StopTaskRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error stopping task: %w", err)
	}

	fmt.Printf("Task %s stopped successfully.\n", taskID)
	return nil
}
