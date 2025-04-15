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
	mu     sync.RWMutex 
	output []byte
	cond   *sync.Cond
	done   chan struct{}
}

// NewTaskWriter initializes a new TaskWriter
func NewTaskWriter() *TaskWriter {
	tw := &TaskWriter{
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
	tw.output = append(tw.output, p...)
	tw.cond.Broadcast()
	tw.mu.Unlock()
	return len(p), nil
}

// maxChunkSize is the maximum number of bytes to send to the client at a time
// TODO: make this configurable
const maxChunkSize = 4096

// ReadOutput reads the output from the task writer. Will send up to maxChunkSize bytes to the client.
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
    for {
        // Quick check using a read lock
        tw.mu.RLock()
        outputLen := int64(len(tw.output))
        if offset < outputLen {
            end := offset + maxChunkSize
            if end > outputLen {
                end = outputLen
            }
            data := slices.Clone(tw.output[offset:end])
            tw.mu.RUnlock()
            return data, end, nil
        }
        tw.mu.RUnlock()

        // cond.Wait() requires a full lock
        tw.mu.Lock()

        select {
        case <-ctx.Done():
            tw.mu.Unlock()
            return nil, offset, ctx.Err()
        case <-tw.done:
            // if there is additional output after task was done return it
            // otherwise return EOF
            if offset >= int64(len(tw.output)) {
                tw.mu.Unlock()
                return nil, offset, io.EOF
            }
        default:
            // Wait for a signal of new output
            tw.cond.Wait()
        }
        tw.mu.Unlock()
    }
}




// Close closes the task writer and wakes up any waiting readers
func (tw *TaskWriter) Close() {
	tw.mu.Lock()
	close(tw.done)
	tw.cond.Broadcast()
	tw.mu.Unlock()
}
