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

func (tr *TaskReader) Close() error {
	// No-op: the context governs cancellation
	return nil
}
