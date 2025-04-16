package task

import (
	"context"
	"io"
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

// GetStreamer returns a reader that reads the output of a task.
// The reader is created with a context that is merged with the client and server contexts.
func (tm *TaskManager) GetStreamer(ctx context.Context, taskID string) (io.ReadCloser, error) {
	taskObj, err := tm.getTaskFromMap(taskID)
	if err != nil {
		return nil, err
	}
	// Merge client and server contexts
	mergedCtx, cancel := mergeCancelContexts(ctx, tm.ctx)

	reader := taskObj.newOutputReader(mergedCtx)
	reader.cancel = cancel
	return reader, nil
}

func (t *Task) newOutputReader(ctx context.Context) *TaskReader {
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
