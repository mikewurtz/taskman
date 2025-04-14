package task

import (
	"context"
	"io"
	"sync"
)

var _ io.Writer = &TaskWriter{}

// TaskWriter handles writing and buffering task output
// Uses a single shared output buffer that is written to for the task
// clients then read from this buffer with their own offsets
type TaskWriter struct {
	mu     sync.RWMutex
	output []byte
	cond   *sync.Cond
	done   chan struct{}
	io.Writer
}

// NewTaskWriter initializes a new TaskWriter
func NewTaskWriter() *TaskWriter {
	tw := &TaskWriter{
        // initialize the output buffer with a default size
        // this prevents us from allocating a new buffer on every write initially
		output: make([]byte, 0, 4096),
		done:   make(chan struct{}),
	}
	tw.cond = sync.NewCond(&tw.mu)
	return tw
}

// Write writes the output to the task writer
// when we append to the buffer we broadcast to wake up any waiting readers
func (tw *TaskWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.output = append(tw.output, p...)
	tw.cond.Broadcast()
	return len(p), nil
}

// Send up to 4KB of data to the client at a 
// TODO: make this configurable
const maxChunkSize = 4096

// ReadOutput reads the output from the task writer. Will send up to maxChunkSize bytes to the client.
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
    tw.mu.Lock()
    defer tw.mu.Unlock()

    for offset >= int64(len(tw.output)) {
        select {
        case <-ctx.Done():
            // if the context is done, then return
            return nil, offset, ctx.Err()
        case <-tw.done:
            // if the task is done and the offset is >= length of output,
            // we are at the end of the output and should return EOF
            if offset >= int64(len(tw.output)) {
                return nil, offset, io.EOF
            }
        default:
            // if we are caught up and the task is not done and the context is not done
            // then wait for the task to write more data
            tw.cond.Wait()
        }
    }

    // Calculate the new offset by reading only up to maxChunkSize bytes
    endOffset := offset + maxChunkSize
    if endOffset > int64(len(tw.output)) {
        endOffset = int64(len(tw.output))
    }
    return tw.output[offset:endOffset], endOffset, nil
}


// Close closes the task writer and wakes up any waiting readers
func (tw *TaskWriter) Close() {
	close(tw.done)
	tw.cond.Broadcast()
}
