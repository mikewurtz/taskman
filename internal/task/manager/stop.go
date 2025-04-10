package task

import (
	"context"
	"syscall"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
)

func (tm *TaskManager) StopTask(ctx context.Context, taskID string) error {
	task, err := tm.GetTask(taskID)
	if err != nil {
		return err
	}

	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if task.ClientID != caller && caller != "admin" {
		return basetask.NewTaskError(basetask.ErrNotFound, "task with id %s not found", taskID)

	}

	if !task.EndTime.IsZero() {
		return basetask.NewTaskError(basetask.ErrFailedPrecondition, "task has already completed")
	}

	if err := syscall.Kill(-task.ProcessID, syscall.SIGKILL); err != nil {
		return basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to send SIGKILL to process group", err)
	}

	task.mu.Lock()
	task.TerminationSource = "user"
	if caller == "admin" {
		task.TerminationSource = "admin"
	}
	task.mu.Unlock()

	return nil
}
