package client

import (
	"context"
	"fmt"
	"time"

	pb "github.com/mikewurtz/taskman/gen/proto"
	"google.golang.org/grpc"
)

// Manager wraps the gRPC client operations
type Manager struct {
	client     pb.TaskManagerClient
	conn       *grpc.ClientConn
}

// NewManager sets up a new gRPC manager
func NewManager(userID, serverAddr string) (*Manager, error) {
	client, conn, err := NewClient(userID, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Manager{
		client:     client,
		conn:       conn,
	}, nil
}

// Close closes the gRPC connection
func (m *Manager) Close() error {
	return m.conn.Close()
}

// StartTask starts a new task with the given command and arguments
func (m *Manager) StartTask(command string, args []string) (string, error) {
	// longer timeout as task may take longer to start. Generally should be much faster than
	// 1 minute, but 1 minute is a safe upper bound
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

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
func (m *Manager) GetTaskStatus(taskID string) error {
	// shorter timeout as status should be quick
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error getting task status: %w", err)
	}

	// TODO handle task status ouput once implemented
	return nil
}

// StreamTaskOutput streams the output of a task by its ID
func (m *Manager) StreamTaskOutput(taskID string) error {
	// context with no timeout because we want to stream indefinitely
	// have a context.WithCancel for clean cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := m.client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error starting output stream: %w", err)
	}

	// TODO handle output stream once implemented
	_, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving from stream: %w", err)
	}

	return nil
}

// StopTask stops a task by its ID
func (m *Manager) StopTask(taskID string) error {
	// 10 second timeout for stopping a task should be a high enough upper bound
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := m.client.StopTask(ctx, &pb.StopTaskRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error stopping task: %w", err)
	}

	fmt.Printf("Task %s stopped successfully.\n", taskID)
	return nil
}
