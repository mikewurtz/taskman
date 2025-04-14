package task

import (
	"context"
	"fmt"
	"io"
	"log"

	basetask "github.com/mikewurtz/taskman/internal/task"
)

func mergeContexts(a, b context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-a.Done():
		case <-b.Done():
		}
		cancel()
	}()
	return ctx, cancel
}

// StreamTaskOutput streams the output of a task to the provided writer function.
// The writer function is called with chunks of output data. Returns when the task is complete or an error occurs.
func (tm *TaskManager) StreamTaskOutput(ctx context.Context, taskID string, writer func([]byte) error) error {
	taskObj, err := tm.getTaskFromMap(taskID)
	if err != nil {
		return err
	}

	// Merge client and server contexts to cancel when either is done
	// if the server context is cancelled, we want to stop the stream
	mergedCtx, cancel := mergeContexts(ctx, tm.ctx)
	defer cancel()

	var offset int64
	for {
		data, newOffset, err := taskObj.ReadOutput(mergedCtx, offset)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if mergedCtx.Err() != nil {
				switch {
				case tm.ctx.Err() != nil:
					log.Printf("Task manager server context canceled: %v", tm.ctx.Err())
					return basetask.NewTaskError(basetask.ErrNotAvailable, "server shutting down")
				default:
					log.Printf("Merged context canceled: %v", mergedCtx.Err())
				}
			}
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
