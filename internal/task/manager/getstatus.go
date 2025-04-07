package task

import (
	"context"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (tm *TaskManager) GetTaskStatus(ctx context.Context, taskID string) (*Task, error) {
	tm.mu.RLock()
	task, ok := tm.tasksMapByID[taskID]
	tm.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "task with id %s not found", taskID)
	}

	caller := ctx.Value(basegrpc.ClientCNKey).(string)
	if task.ClientID != caller && caller != "admin" {
		return nil, status.Errorf(codes.NotFound, "task with id %s not found", taskID)
	}

	task.mu.RLock()
	defer task.mu.RUnlock()

	copy := &Task{
		ID:                task.ID,
		ClientID:          task.ClientID,
		ProcessID:         task.ProcessID,
		Status:            task.Status,
		StartTime:         task.StartTime,
		EndTime:           task.EndTime,
		TerminationSignal: task.TerminationSignal,
		TerminationSource: task.TerminationSource,
	}
	if task.ExitCode != nil {
		ec := *task.ExitCode
		copy.ExitCode = &ec
	}
	return copy, nil
}

