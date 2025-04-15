package task

import (
	"context"
	"errors"
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

	// Merge client and server contexts
	mergedCtx, cancel := mergeCancelContexts(ctx, tm.ctx)
	defer cancel()

	reader := taskObj.newOutputReader(mergedCtx)

	buf := make([]byte, maxChunkSize)

	for {
		n, err := reader.Read(buf)
		if errors.Is(err, io.EOF) {
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

		if n > 0 {
			if err := writer(buf[:n]); err != nil {
				return basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to write output", err)
			}
		}
	}
}

func (t *Task) newOutputReader(ctx context.Context) io.ReadCloser {
	return &TaskReader{
		tw:  t.getWriter(),
		ctx: ctx,
	}
}

func (t *Task) getWriter() *TaskWriter {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.writer
}
