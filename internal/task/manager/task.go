package task

import (
	"context"
	"sync"
	"time"

	basetask "github.com/mikewurtz/taskman/internal/task"
)

// Task represents a managed process. All fields are protected by the mu mutex.
// The Task struct is safe for concurrent access through its methods.
// Direct field access is not allowed; all access must be through getters and setters.
type Task struct {
	mu                sync.RWMutex
	id                string
	clientID          string
	processID         int
	status            int
	startTime         time.Time
	exitCode          *int32
	terminationSignal string
	terminationSource string
	endTime           time.Time
	done              chan struct{}

	writer *TaskWriter
}

// NewTask creates a new task
func CreateNewTask(id, clientID string, pid int, startTime time.Time, writer *TaskWriter) *Task {
	t := &Task{
		id:        id,
		clientID:  clientID,
		processID: pid,
		startTime: startTime,
		status:    basetask.JobStatusStarted,
		done:      make(chan struct{}),
		writer:    writer,
	}
	return t
}

func (t *Task) GetWriter() *TaskWriter {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.writer
}

// Update ReadOutput to delegate to TaskWriter
func (t *Task) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
    return t.writer.ReadOutput(ctx, offset)
}

// GetID returns the task ID
func (t *Task) GetID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

// GetClientID returns the client ID
func (t *Task) GetClientID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.clientID
}

// GetProcessID returns the process group ID
func (t *Task) GetProcessID() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.processID
}

// GetStatus returns the task status.
func (t *Task) GetStatus() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetStartTime returns the task's start time
func (t *Task) GetStartTime() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.startTime
}

// GetEndTime returns the task's end time
func (t *Task) GetEndTime() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.endTime
}

// GetExitCode returns a copy of the exit code pointer
func (t *Task) GetExitCode() *int32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.exitCode == nil {
		return nil
	}
	val := *t.exitCode
	return &val
}

// GetTerminationSignal returns the termination signal
func (t *Task) GetTerminationSignal() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.terminationSignal
}

// GetTerminationSource returns the source of termination
func (t *Task) GetTerminationSource() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.terminationSource
}

// SetStatus updates the task status.
func (t *Task) SetStatus(status int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = status
}

// SetExitCode sets the exit code.
func (t *Task) SetExitCode(code *int32) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.exitCode = code
}

// SetTerminationSignal sets the termination signal string.
func (t *Task) SetTerminationSignal(sig string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.terminationSignal = sig
}

// SetTerminationSource sets the termination source.
func (t *Task) SetTerminationSource(source string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.terminationSource = source
}

// SetEndTime sets the task's end time.
func (t *Task) SetEndTime(tstamp time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.endTime = tstamp
}

// Done returns a read-only channel that is closed when the task completes
func (t *Task) Done() <-chan struct{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.done
}
