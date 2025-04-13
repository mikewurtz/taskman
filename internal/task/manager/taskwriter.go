package task

import (
	"context"
	"io"
	"sync"
)

var _ io.Writer = &TaskWriter{}

// TaskWriter handles writing and buffering task output
type TaskWriter struct {
	mu     sync.RWMutex
	output []byte
	cond   *sync.Cond
	done   chan struct{}
	io.Writer
}

func NewTaskWriter() *TaskWriter {
	tw := &TaskWriter{
		output: make([]byte, 0, 4096),
		done:   make(chan struct{}),
	}
	tw.cond = sync.NewCond(&tw.mu)
	return tw
}

// Write writes the output to the task writer
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

// ReadOutput reads the output from the task writer
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
    tw.mu.Lock()
    defer tw.mu.Unlock()

    for offset >= int64(len(tw.output)) {
        select {
        case <-ctx.Done():
            return nil, offset, ctx.Err()
        case <-tw.done:
            if offset >= int64(len(tw.output)) {
                return nil, offset, io.EOF
            }
        default:
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
