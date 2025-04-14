package task

import (
	pb "github.com/mikewurtz/taskman/gen/proto"
)

// Internal job status constants
const (
	JobStatusUnknown = iota
	JobStatusStarted
	JobStatusSignaled
	JobStatusExitedOK
	JobStatusExitedError
)

// StatusToProto converts internal status strings to proto JobStatus enum
func StatusToProto(internal int) (pb.JobStatus, error) {
	switch internal {
	case JobStatusUnknown:
		return pb.JobStatus_JOB_STATUS_UNKNOWN, nil
	case JobStatusStarted:
		return pb.JobStatus_JOB_STATUS_STARTED, nil
	case JobStatusSignaled:
		return pb.JobStatus_JOB_STATUS_SIGNALED, nil
	case JobStatusExitedOK:
		return pb.JobStatus_JOB_STATUS_EXITED_OK, nil
	case JobStatusExitedError:
		return pb.JobStatus_JOB_STATUS_EXITED_ERROR, nil
	default:
		return pb.JobStatus_JOB_STATUS_UNKNOWN, NewTaskError(ErrInternal, "unknown internal job status: %q", internal)
	}
}
