package task

import (
	"context"
	"fmt"
	"syscall"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
)

func (tm *TaskManager) StopTask(ctx context.Context, taskID string) error {
	tm.mu.RLock()
	task, ok := tm.tasksMapByID[taskID]
	tm.mu.RUnlock()
	if !ok {
		// TODO do I need the %s here?
		return basetask.NewTaskError(basetask.ErrNotFound, "%s", fmt.Sprintf("task with id %s not found", taskID))
	}

	caller := ctx.Value(basegrpc.ClientIDKey).(string)
	if task.ClientID != caller && caller != "admin" {
		// TODO do I need the %s here?
		return basetask.NewTaskError(basetask.ErrNotFound, "%s", fmt.Sprintf("task with id %s not found", taskID))
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