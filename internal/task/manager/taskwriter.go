package task

import (
	"context"
	"io"
	"slices"
	"sync"
)

// make sure TaskWriter implements io.Writer
var _ io.Writer = &TaskWriter{}

// TaskWriter handles writing and buffering task output
// Uses a single shared output buffer that is written to for the task
// clients then read from this buffer with their own offsets
type TaskWriter struct {
	// protects access to output buffer
	outputMu sync.RWMutex
	output   []byte

	// protects condition variable
	condMu sync.Mutex
	cond   *sync.Cond
	done   chan struct{}
}

// NewTaskWriter initializes a new TaskWriter
func NewTaskWriter() *TaskWriter {
	tw := &TaskWriter{
		output: make([]byte, 0, 4096),
		done:   make(chan struct{}),
	}
	tw.cond = sync.NewCond(&tw.condMu)
	return tw
}

// Write writes the output to the task writer
// when we append to the buffer we broadcast to wake up any waiting readers
func (tw *TaskWriter) Write(p []byte) (n int, err error) {
	tw.outputMu.Lock()
	tw.output = append(tw.output, p...)
	tw.outputMu.Unlock()

	tw.condMu.Lock()
	tw.cond.Broadcast()
	tw.condMu.Unlock()

	return len(p), nil
}

// maxChunkSize is the maximum number of bytes to send to the client at a time
// TODO: make this configurable
const maxChunkSize = 4096

// ReadOutput reads the output from the task writer. Will send up to maxChunkSize bytes to the client.
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
	for {
		tw.outputMu.RLock()
		if offset < int64(len(tw.output)) {
			end := offset + maxChunkSize
			if end > int64(len(tw.output)) {
				end = int64(len(tw.output))
			}
            // return a clone of the data
			data := slices.Clone(tw.output[offset:end])
			tw.outputMu.RUnlock()
			return data, end, nil
		}
		tw.outputMu.RUnlock()

		select {
		case <-ctx.Done():
			return nil, offset, ctx.Err()
		case <-tw.done:
			tw.outputMu.RLock()
			if offset >= int64(len(tw.output)) {
				tw.outputMu.RUnlock()
				return nil, offset, io.EOF
			}
			tw.outputMu.RUnlock()
		default:
			// wait for the condition variable to be signaled
			tw.condMu.Lock()
			tw.cond.Wait()
			tw.condMu.Unlock()
		}
	}
}

// Close closes the task writer and wakes up any waiting readers
func (tw *TaskWriter) Close() {
	tw.condMu.Lock()
	close(tw.done)
	tw.cond.Broadcast()
	tw.condMu.Unlock()
}
