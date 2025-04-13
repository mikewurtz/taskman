package task

import (
	"context"
	"fmt"
	"io"
)

// StreamTaskOutput streams the output of a task to the provided writer function.
// The writer function is called with chunks of output data. Returns when the task is complete or an error occurs.
func (tm *TaskManager) StreamTaskOutput(ctx context.Context, taskID string, writer func([]byte) error) error {
	taskObj, err := tm.getTaskFromMap(taskID)
	if err != nil {
		return err
	}
	
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		data, newOffset, err := taskObj.ReadOutput(ctx, offset)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read output: %w", err)
		}

		if len(data) > 0 {
			if err := writer(data); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		}

		offset = newOffset
	}
}
