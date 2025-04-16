package task

import (
	"context"
	"io"
)

// make sure TaskReader implements io.Reader
var _ io.ReadCloser = &TaskReader{}

// TaskReader reads the output of a task
type TaskReader struct {
	tw     *TaskWriter
	ctx    context.Context
	offset int64
	cancel context.CancelFunc
}

// Read reads the output of the task
func (tr *TaskReader) Read(p []byte) (int, error) {
	data, nextOffset, err := tr.tw.ReadOutput(tr.ctx, tr.offset)
	if err != nil {
		return 0, err
	}
	n := copy(p, data)
	tr.offset = nextOffset
	return n, nil
}

// Close should now cancel the underlying context to stop further reads.
func (tr *TaskReader) Close() error {
	if tr.cancel != nil {
		tr.cancel()
	}
	return nil
}
