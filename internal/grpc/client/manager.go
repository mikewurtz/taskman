package client

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/mikewurtz/taskman/gen/proto"
	
)

// Manager wraps the gRPC client operations
type Manager struct {
	userID     string
	serverAddr string
}

// NewManager creates a new task manager client
func NewManager(userID, serverAddr string) *Manager {
	return &Manager{
		userID:     userID,
		serverAddr: serverAddr,
	}
}

// StartTask starts a new task with the given command and arguments
func (m *Manager) StartTask(command string, args []string) (string, error) {
	client, conn, err := NewClient(m.userID, m.serverAddr)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	resp, err := client.StartTask(ctx, &pb.StartTaskRequest{
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
	client, conn, err := NewClient(m.userID, m.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error getting task status: %w", err)
	}

	fmt.Printf("Task %s running: %v, process ID: %s, exit code: %d\n",
		taskID, resp.Status, resp.ProcessId, resp.ExitCode)
	return nil
}

// StreamTaskOutput streams the output of a task by its ID
func (m *Manager) StreamTaskOutput(taskID string) error {
	client, conn, err := NewClient(m.userID, m.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stream, err := client.StreamTaskOutput(ctx, &pb.StreamTaskOutputRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error starting output stream: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Stream closed by server.")
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("error receiving task output: %w", err)
		}
		fmt.Println(msg.Output)
	}
}

// StopTask stops a task by its ID
func (m *Manager) StopTask(taskID string) error {
	client, conn, err := NewClient(m.userID, m.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.StopTask(ctx, &pb.StopTaskRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error stopping task: %w", err)
	}

	fmt.Printf("Task %s stopped successfully.\n", taskID)
	return nil
}
