package task

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	// ErrFailedPrecondition indicates that the task is in a failed precondition state
	ErrFailedPrecondition
	// ErrInternal indicates an internal system error
	ErrInternal
	// ErrNotAvailable indicates that the requested resource is not available
	ErrNotAvailable
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

// TaskErrorToGRPC converts a TaskError to a gRPC status error
func TaskErrorToGRPC(err error) error {
	var taskErr *TaskError
	if errors.As(err, &taskErr) {
		var code codes.Code
		switch taskErr.Code {
		case ErrInvalidArgument:
			code = codes.InvalidArgument
		case ErrNotFound:
			code = codes.NotFound
		case ErrAlreadyTerminated:
			code = codes.FailedPrecondition
		case ErrFailedPrecondition:
			code = codes.FailedPrecondition
		case ErrInternal:
			code = codes.Internal
		case ErrNotAvailable:
			code = codes.Unavailable
		default:
			code = codes.Internal
		}
		return status.Error(code, taskErr.Error())
	}
	// If it's not a TaskError, return it as an internal error
	return status.Error(codes.Internal, err.Error())
}
