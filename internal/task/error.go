package task

import (
	"errors"
	"fmt"
)

// ErrorCode represents the type of error that occurred
type ErrorCode int

const (
	// ErrInvalidArgument indicates that the provided arguments are invalid
	ErrInvalidArgument ErrorCode = iota
	// ErrNotFound indicates that the requested resource was not found
	ErrNotFound
	// ErrAlreadyTerminated indicates that the task has already terminated
	ErrAlreadyTerminated
	// ErrInternal indicates an internal system error
	ErrInternal
)

// TaskError represents an error that occurred during task management
type TaskError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *TaskError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *TaskError) Unwrap() error {
	return e.Err
}

// NewTaskError creates a new TaskError with optional formatting
func NewTaskError(code ErrorCode, format string, args ...any) *TaskError {
	return &TaskError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// NewTaskErrorWithErr creates a new TaskError with an underlying error
func NewTaskErrorWithErr(code ErrorCode, format string, err error, args ...any) *TaskError {
	return &TaskError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// IsTaskError checks if an error is a TaskError
func IsTaskError(err error) (*TaskError, bool) {
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		return taskErr, true
	}
	return nil, false
}
