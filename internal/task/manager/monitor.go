package task

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/mikewurtz/taskman/internal/task/cgroups"

	basetask "github.com/mikewurtz/taskman/internal/task"
)

// monitorProcess handles the process completion and status updates
func (tm *TaskManager) monitorProcess(taskID string, cmd *exec.Cmd) {
	// Create a channel to receive the process completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for either the process to complete or the context to be canceled
	var cmdErr error
	select {
	case cmdErr = <-done:
		// Process completed normally
	case <-tm.ctx.Done():
		// Server context was canceled, kill the entire process group
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("Failed to kill process group %d: %v", cmd.Process.Pid, err)
		}
		cmdErr = <-done
	}

	finishTime := time.Now()

	if cmdErr != nil {
		log.Printf("Task %s exited with an error: %v", taskID, cmdErr)
	} else {
		log.Printf("Task %s completed successfully", taskID)
	}

	var exitCode *int
	var signal string
	exitCode, signal = extractProcessExitInfo(cmdErr, cmd)

	task, err := tm.GetTask(taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
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

	if oomKilled, err := cgroups.CheckIfOOMKilled(taskID); err != nil {
		log.Printf("Failed to check if task %s was OOM killed: %v", taskID, err)
	} else if oomKilled {
		// OOM kill overrides whatever status was previously inferred
		task.Status = basetask.JobStatusSignaled
		task.TerminationSignal = syscall.SIGKILL.String()
		task.TerminationSource = "oom"
		task.ExitCode = nil
	} else if exitCode != nil {
		if *exitCode == 0 {
			task.Status = basetask.JobStatusExitedOK
		} else {
			task.Status = basetask.JobStatusExitedError
		}
	} else {
		task.Status = basetask.JobStatusSignaled
		if task.TerminationSource == "" {
			task.TerminationSource = "system"
		}
	}

	// Clean up cgroup after process completes
	if cleanupErr := cgroups.RemoveCgroupForTask(taskID); cleanupErr != nil {
		log.Printf("Failed to clean up cgroup after process completion: %v", cleanupErr)
	}

	// Signal that this task is done
	task.doOnce.Do(func() {
		close(task.done)
	})
}

// extractProcessExitInfo extracts the exit code and signal from the command error
// or from the process state if the command terminated normally
func extractProcessExitInfo(cmdErr error, cmd *exec.Cmd) (*int, string) {
	var exitCode *int
	var signal string

	if cmdErr != nil {
		// Handle context-related errors
		if errors.Is(cmdErr, context.DeadlineExceeded) {
			log.Printf("Command timed out: %v", cmdErr)
		} else if errors.Is(cmdErr, context.Canceled) {
			log.Printf("Command context was canceled: %v", cmdErr)
		}

		// os.PathError and exec.Error are handled when we call cmd.Start()
		switch e := cmdErr.(type) {
		case *exec.ExitError:
			// Process started but exited with non-zero status or was killed
			if status, ok := e.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					signal = status.Signal().String()
				} else {
					code := status.ExitStatus()
					exitCode = &code
				}
			} else {
				log.Printf("Unexpected type in ExitError.Sys(): %T", e.Sys())
			}
		default:
			log.Printf("Unhandled command error type (%T): %v", cmdErr, cmdErr)
		}
	} else {
		// process exited with no error
		if cmd.ProcessState == nil {
			log.Printf("Missing ProcessState for completed process, cannot extract exit info")
			return nil, ""
		}
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				signal = status.Signal().String()
			} else {
				code := status.ExitStatus()
				exitCode = &code
			}
		} else {
			log.Printf("Unexpected type in ProcessState.Sys(): %T", cmd.ProcessState.Sys())
		}
	}

	return exitCode, signal
}
