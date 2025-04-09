package task

import (
	"context"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/google/uuid"

	basegrpc "github.com/mikewurtz/taskman/internal/grpc"
	basetask "github.com/mikewurtz/taskman/internal/task"
	"github.com/mikewurtz/taskman/internal/task/cgroups"
)

func (tm *TaskManager) StartTask(ctx context.Context, command string, args []string) (string, error) {
	clientCN := ctx.Value(basegrpc.ClientCNKey)
	log.Printf("Starting task for client %s: %s %v", clientCN, command, args)

	if command == "" {
		return "", basetask.NewTaskError(basetask.ErrInvalidArgument, "command cannot be empty")
	}

	taskID := uuid.New().String()

	// Create cgroup and get file descriptor
	cgroupFd, err := cgroups.CreateCgroupForTask(taskID)
	if err != nil {
		return "", basetask.NewTaskError(basetask.ErrInternal, "failed to create cgroup")
	}

	cmd := exec.Command(command, args...)

	// Set process attributes. We set the cgroup fields so the process starts in the cgroup rather than having to move it later
	// We want the pgid so we can kill the entire process group later
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:     true,
		UseCgroupFD: true,
		CgroupFD:    int(cgroupFd.Fd()),
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		// clean up the cgroup so it doesn't leak
		cgroupFd.Close()
		if cleanupErr := cgroups.RemoveCgroupForTask(taskID); cleanupErr != nil {
			log.Printf("Failed to clean up cgroup after process start failure: %v", cleanupErr)
		}
		switch e := err.(type) {
		case *exec.Error:
			return "", basetask.NewTaskErrorWithErr(basetask.ErrInvalidArgument, "invalid command", e)
		case *os.PathError:
			return "", basetask.NewTaskErrorWithErr(basetask.ErrInvalidArgument, "command not found or not executable", e)
		default:
			return "", basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to start process", err)
		}
	}

	// Get the process group ID
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// if we fail to get pgid fallback to PID
		pgid = cmd.Process.Pid
	}

	// we can now safely close the CgroupFD
	cgroupFd.Close()

	task := &Task{
		ID:        taskID,
		StartTime: time.Now(),
		ClientID:  clientCN.(string),
		Status:    "JOB_STATUS_STARTED",
		ProcessID: pgid,
	}

	tm.AddTask(task)

	// TODO move this goroutine into its own function
	go func(taskID string, cmd *exec.Cmd) {
		err := cmd.Wait()
		finishTime := time.Now()

		var exitCode int
		var signal string

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
					if status.Signaled() {
						signal = status.Signal().String()
					} else {
						// TODO why is this negative 1?
						log.Println("exit code", status.ExitStatus())
						exitCode = status.ExitStatus()
					}
				}
			}
		} else {
			if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}

		tm.mu.RLock()
		task, ok := tm.tasksMapByID[taskID]
		tm.mu.RUnlock()
		if !ok {
			return
		}

		task.mu.Lock()
		defer task.mu.Unlock()

		task.EndTime = finishTime
		ec := int32(exitCode)
		task.ExitCode = &ec
		task.TerminationSignal = signal

		if err != nil {
			log.Printf("Task %s failed: %v", taskID, err)
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
	}(taskID, cmd)

	return taskID, nil
}