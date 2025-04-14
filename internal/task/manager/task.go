package task

import (
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
}

// TaskSnapshot is a snapshot of the task's state
// used to return task information to reduce the number of lock calls
type TaskSnapshot struct {
	ID                string
	ClientID          string
	ProcessID         int
	Status            int
	StartTime         time.Time
	EndTime           time.Time
	ExitCode          *int32
	TerminationSignal string
	TerminationSource string
}

// NewTask creates a new task
func CreateNewTask(id, clientID string, pid int, startTime time.Time) *Task {
	return &Task{
		id:        id,
		clientID:  clientID,
		processID: pid,
		startTime: startTime,
		status:    basetask.JobStatusStarted,
		done:      make(chan struct{}),
	}
}

// GetID returns the task ID
func (t *Task) GetID() string {
	return t.id
}

// GetClientID returns the client ID
func (t *Task) GetClientID() string {
	return t.clientID
}

// GetProcessID returns the process group ID
func (t *Task) GetProcessID() int {
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

// Snapshot returns a snapshot of the task's state
// used to reduce the number of lock calls
func (t *Task) Snapshot() TaskSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var exitCodeCopy *int32
	if t.exitCode != nil {
		val := *t.exitCode
		exitCodeCopy = &val
	}

	return TaskSnapshot{
		ID:                t.id,
		ClientID:          t.clientID,
		ProcessID:         t.processID,
		Status:            t.status,
		StartTime:         t.startTime,
		EndTime:           t.endTime,
		ExitCode:          exitCodeCopy,
		TerminationSignal: t.terminationSignal,
		TerminationSource: t.terminationSource,
	}
}
