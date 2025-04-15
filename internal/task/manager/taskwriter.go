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
)

// ReadOutput reads the output from the task writer. Will send up to maxChunkSize bytes to the client.
// It returns the next offset to read from and the data read. If there is no more data to read it returns io.EOF.
// If the context is cancelled it returns the context error
func (tw *TaskWriter) ReadOutput(ctx context.Context, offset int64) ([]byte, int64, error) {
	stopWake := context.AfterFunc(ctx, func() {
		tw.mu.Lock()
		tw.cond.Broadcast()
		tw.mu.Unlock()
	})
	defer stopWake()

	// Fast path with read lock
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
	defer tw.mu.Unlock()

	for {
		outputLen = int64(len(tw.output))
		if offset < outputLen {
			end := offset + maxChunkSize
			if end > outputLen {
				end = outputLen
			}
			data := slices.Clone(tw.output[offset:end])
			return data, end, nil
		}

		select {
		case <-ctx.Done():
			return nil, offset, ctx.Err()
		case <-tw.done:
			if offset >= int64(len(tw.output)) {
				return nil, offset, io.EOF
			}
		default:
		}

		tw.cond.Wait()
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
