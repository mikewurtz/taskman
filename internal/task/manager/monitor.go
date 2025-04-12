package task

import (
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
	errC := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for either the process to complete or the context to be canceled
	var cmdErr error
	select {
	case cmdErr = <-done:
		// Process completed either normally or with an error
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

	unknownStatus := false
	if exitCode == nil && signal == "" {
		// Unknown failure — ProcessState or WaitStatus was missing or corrupt
		task.Status = basetask.JobStatusUnknown
		task.TerminationSource = "unknown"
		log.Printf("Could not determine how task %s exited", task.ID)
		unknownStatus = true
	}

	task.EndTime = finishTime
	if exitCode != nil {
		ec := int32(*exitCode)
		task.ExitCode = &ec
	}
	task.TerminationSignal = signal

	if !unknownStatus {
		if oomKilled, err := cgroups.CheckIfOOMKilled(taskID); err != nil {
			log.Printf("Failed to check if task %s was OOM killed: %v", taskID, err)
		} else if oomKilled {
			// OOM kill overrides whatever status was previously inferred.
			// Because we monitor the process group ID, the kernel may have killed a child
			// process instead. In that case, the PGID process may exit with code 1,
			// which would incorrectly appear as a regular failure.
			// To reflect the true cause, we override the status and clear ExitCode.
			log.Printf("Task %s was OOM killed; overriding status to SIGKILL", task.ID)
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
	var ws syscall.WaitStatus
	var ok bool

	switch {
	case cmdErr != nil:
		// exec.Error and exec.PathError should already be handled on cmd.Start()
		switch e := cmdErr.(type) {
		case *exec.ExitError:
			ws, ok = e.Sys().(syscall.WaitStatus)
			if !ok {
				log.Printf("Unexpected type in ExitError.Sys(): %T", e.Sys())
				return nil, ""
			}
		default:
			log.Printf("Unhandled command error type (%T): %v", cmdErr, cmdErr)
			return nil, ""
		}
	default:
		if cmd.ProcessState == nil {
			log.Printf("Missing ProcessState for completed process, cannot extract exit info")
			return nil, ""
		}
		ws, ok = cmd.ProcessState.Sys().(syscall.WaitStatus)
		if !ok {
			log.Printf("Unexpected type in ProcessState.Sys(): %T", cmd.ProcessState.Sys())
			return nil, ""
		}
	}

	// Extract signal or exit code
	if ws.Signaled() {
		signal = ws.Signal().String()
	} else {
		code := ws.ExitStatus()
		exitCode = &code
	}

	return exitCode, signal
}
