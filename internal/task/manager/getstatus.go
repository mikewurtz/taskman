package task

import (
	"context"
)

// GetTaskStatus returns the task object to be used to show task status
func (tm *TaskManager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return tm.getTaskFromMap(taskID)
}
