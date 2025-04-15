package task

import (
	"context"
	"io"
	"slices"
	"sync"
	"time"
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
	once   sync.Once
}

// NewTaskWriter initializes a new TaskWriter
func NewTaskWriter() *TaskWriter {
	tw := &TaskWriter{
		// TODO: make this configurable
		// Set up a buffer with an initial size so we avoid reallocations
		// early on when the task is just starting
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

// TODO: make these configurable
const (
	// maxChunkSize is the maximum number of bytes to send to the client at a time
	maxChunkSize = 4096
	// recheckInterval is the interval to recheck the context
	recheckInterval = 500 * time.Millisecond
)

// ReadOutput reads the output from the task writer. Will send up to maxChunkSize bytes to the client.
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
	for {
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

		tw.mu.Lock()
		// set up a timer to recheck if the context has been cancelled
		timer := time.NewTimer(recheckInterval)

		select {
		case <-ctx.Done():
			tw.mu.Unlock()
			timer.Stop()
			return nil, offset, ctx.Err()
		case <-tw.done:
			if offset >= int64(len(tw.output)) {
				tw.mu.Unlock()
				timer.Stop()
				return nil, offset, io.EOF
			}
		case <-timer.C:
			// Just fall through and recheck output
		}

		timer.Stop()
		tw.mu.Unlock()
	}
}

// Close closes the task writer and wakes up any waiting readers
func (tw *TaskWriter) Close() {
	tw.once.Do(func() {
		tw.mu.Lock()
		defer tw.mu.Unlock()
		close(tw.done)
		tw.cond.Broadcast()
	})
}
