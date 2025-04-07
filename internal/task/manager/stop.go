package task


import (
	"context"
	"syscall"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (tm *TaskManager) StopTask(ctx context.Context, taskID string) error {
	tm.mu.RLock()
	task, ok := tm.tasksMapByID[taskID]
	tm.mu.RUnlock()
	if !ok {
		return status.Errorf(codes.NotFound, "task with id %s not found", taskID)
	}

	caller := ctx.Value(basegrpc.ClientCNKey).(string)
	if task.ClientID != caller && caller != "admin" {
		return status.Errorf(codes.NotFound, "task with id %s not found", taskID)
	}

	if err := syscall.Kill(-task.ProcessID, syscall.SIGTERM); err != nil {
		return basetask.NewTaskError(codes.Internal, "failed to send SIGTERM to process group", err)
	}

	task.mu.Lock()
	task.TerminationSource = "user"
	task.mu.Unlock()

	return nil
}