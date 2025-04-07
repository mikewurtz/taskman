package task

import (
	"google.golang.org/grpc/codes"
)

// TaskError represents an error that occurred during task management
type TaskError struct {
	Code    codes.Code
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

// NewTaskError creates a new TaskError
func NewTaskError(code codes.Code, message string, err error) *TaskError {
	return &TaskError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
