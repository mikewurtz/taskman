package task

import (
	"context"
	"io"
	"log"

	basetask "github.com/mikewurtz/taskman/internal/task"
)

// mergeCancelContexts merges two contexts and cancels when either is done
func mergeCancelContexts(a, b context.Context) (context.Context, context.CancelFunc) {
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
	// if the either the client or server context is cancelled, we want to stop the stream
	mergedCtx, cancel := mergeCancelContexts(ctx, tm.ctx)
	defer cancel()

	var offset int64
	for {
		data, newOffset, err := taskObj.ReadOutput(mergedCtx, offset)
		if err == io.EOF {
			// we hit end of the output stream; return nil to indicate success
			return nil
		}
		if err != nil {
			if mergedCtx.Err() != nil {
				switch {
				case tm.ctx.Err() != nil:
					log.Printf("Task manager server context canceled: %v", tm.ctx.Err())
					return basetask.NewTaskError(basetask.ErrNotAvailable, "server shutting down")
				default:
					log.Printf("Client context canceled: %v", mergedCtx.Err())
					return basetask.NewTaskError(basetask.ErrCanceled, "client canceled stream")
				}
			}
			return basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to read output", err)
		}

		if len(data) > 0 {
			// write the data to the provided writer function
			if err := writer(data); err != nil {
				return basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to write output", err)
			}
		}

		offset = newOffset
	}
}
