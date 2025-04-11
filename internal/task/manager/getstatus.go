package task

import (
	"context"
	"fmt"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
)

func (tm *TaskManager) GetTaskStatus(ctx context.Context, taskID string) (*Task, error) {
	task, err := tm.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if task.ClientID != caller && caller != "admin" {
		return nil, basetask.NewTaskError(basetask.ErrNotFound, fmt.Sprintf("task with id %s not found", taskID), nil)
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
