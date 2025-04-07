package client

import (
	"bytes"
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	pb "github.com/mikewurtz/taskman/gen/proto"
	"google.golang.org/grpc"
)

// Manager wraps the gRPC client operations
type Manager struct {
	client pb.TaskManagerClient
	conn   *grpc.ClientConn
}

// NewManager sets up a new gRPC manager
func NewManager(userID, serverAddr string) (*Manager, error) {
	client, conn, err := NewClient(userID, serverAddr)
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

type TaskStatus struct {
	TaskID            string
	Status            string
	StartTime         time.Time
	EndTime           time.Time
	ExitCode          *int32
	ProcessID         int32
	TerminationSignal string
	TerminationSource string
}

func (t *TaskStatus) String() string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTART TIME\tPID\tSTATUS\tEXIT CODE\tSIGNAL\tSOURCE\tEND TIME")
	fmt.Fprintln(w, "-------\t----------\t---\t------\t---------\t------\t------\t--------")

	startTime := t.StartTime.Format("2006-01-02 15:04:05")

	endTime := "-"
	if !t.EndTime.IsZero() {
		endTime = t.EndTime.Format("2006-01-02 15:04:05")
	}

	exitStr := "-"
	if t.ExitCode != nil {
		exitStr = fmt.Sprintf("%d", *t.ExitCode)
	}

	signal := t.TerminationSignal
	if signal == "" {
		signal = "-"
	}

	source := t.TerminationSource
	if source == "" {
		source = "-"
	}

	fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
		t.TaskID,
		startTime,
		t.ProcessID,
		t.Status,
		exitStr,
		signal,
		source,
		endTime,
	)
	w.Flush()
	return buf.String()
}

// GetTaskStatus gets the status of a task by its ID
func (m *Manager) GetTaskStatus(taskID string) (*TaskStatus, error) {
	// shorter timeout as status should be quick
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	// StopTask response is an empty object
	_, err := m.client.StopTask(ctx, &pb.StopTaskRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("error stopping task: %w", err)
	}

	fmt.Printf("Task %s stopped successfully.\n", taskID)
	return nil
}
