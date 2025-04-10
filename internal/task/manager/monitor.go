package task

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/mikewurtz/taskman/internal/task/cgroups"
)

// monitorProcess handles the process completion and status updates
func (tm *TaskManager) monitorProcess(taskID string, cmd *exec.Cmd) {
	cmdErr := cmd.Wait()
	finishTime := time.Now()

	var exitCode *int
	var signal string

	exitCode, signal = extractProcessExitInfo(cmdErr, cmd)

	tm.mu.RLock()
	task, ok := tm.tasksMapByID[taskID]
	tm.mu.RUnlock()
	if !ok {
		return
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	task.EndTime = finishTime
	if exitCode != nil {
		ec := int32(*exitCode)
		task.ExitCode = &ec
	}
	task.TerminationSignal = signal

	if cmdErr != nil {
		log.Printf("Task %s failed: %v", taskID, cmdErr)
		if task.TerminationSource == "" && signal == "" {
			task.Status = "JOB_STATUS_EXITED_ERROR"
		} else {
			if task.TerminationSource == "" {
				task.TerminationSource = "external"
			}
			task.Status = "JOB_STATUS_SIGNALED"
		}
	} else {
		task.Status = "JOB_STATUS_EXITED_OK"
		log.Printf("Task %s completed successfully", taskID)
	}

	// Clean up cgroup after process completes
	if cleanupErr := cgroups.RemoveCgroupForTask(taskID); cleanupErr != nil {
		log.Printf("Failed to clean up cgroup after process completion: %v", cleanupErr)
	}
}

// extractProcessExitInfo extracts the exit code and signal from the command error
// or from the process state if the command terminated normally
func extractProcessExitInfo(cmdErr error, cmd *exec.Cmd) (*int, string) {
	var exitCode *int
	var signal string
	if cmdErr != nil {
		// Handle context related errors
		if errors.Is(cmdErr, context.DeadlineExceeded) {
			log.Printf("Command timed out: %v", cmdErr)
			// possibly record or handle this as a timeout error
		} else if errors.Is(cmdErr, context.Canceled) {
			log.Printf("Command context was canceled: %v", cmdErr)
			// handle cancellation as appropriate
		} else {
			// Now do a type switch for more specific error types.
			switch e := cmdErr.(type) {
			case *exec.ExitError:
				// Process started but exited with a nonzero status or was killed.
				if status, ok := e.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						signal = status.Signal().String()
					} else {
						code := status.ExitStatus()
						exitCode = &code
					}
				} else {
					log.Println("Failed to extract wait status from exec.ExitError")
				}
			case *os.PathError:
				// Process failed to start: this could mean the executable wasn't found
				log.Printf("Process failed to start (os.PathError): %v", e)
			case *exec.Error:
				log.Printf("Error executing command: %v", e)
			default:
				log.Printf("Unexpected error type (%T): %v", e, e)
			}
		}
	} else {
		// cmdErr is nil, so the process terminated normally
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				signal = status.Signal().String()
			} else {
				code := status.ExitStatus()
				exitCode = &code
			}
		} else {
			log.Printf("Failed to extract wait status from ProcessState")
		}
	}
	return exitCode, signal
}