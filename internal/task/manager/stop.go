package task

import (
	"context"
	"syscall"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
)

// StopTask stops a task by sending a SIGKILL to the process group
func (tm *TaskManager) StopTask(ctx context.Context, taskID string) error {
	task, err := tm.getTaskFromMap(taskID)
	if err != nil {
		return err
	}

	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if task.GetClientID() != caller && caller != "admin" {
		return basetask.NewTaskError(basetask.ErrNotFound, "task with id %s not found", taskID)

	}

	alreadyCompleted := !task.GetEndTime().IsZero()

	if alreadyCompleted {
		return basetask.NewTaskError(basetask.ErrFailedPrecondition, "task has already completed")
	}

	if err := syscall.Kill(-task.GetProcessID(), syscall.SIGKILL); err != nil {
		return basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to send SIGKILL to process group", err)
	}

	task.SetTerminationSource("user")
	if caller == "admin" {
		task.SetTerminationSource("admin")
	}

	return nil
}
