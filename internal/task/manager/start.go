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

// StartTask starts a new task with the given command and arguments
func (tm *TaskManager) StartTask(ctx context.Context, command string, args []string) (string, error) {
	clientID := ctx.Value(basegrpc.ClientIDKey)
	log.Printf("Starting task for client %s: %s %v", clientID, command, args)

	if command == "" {
		return "", basetask.NewTaskError(basetask.ErrInvalidArgument, "command cannot be empty")
	}

	taskID := uuid.New().String()

	// Create cgroup and get file descriptor
	cgroupFd, err := cgroups.CreateCgroupForTask(taskID)
	if err != nil {
		// if we fail to create the cgroup, try to remove it
		err = cgroups.RemoveCgroupForTask(taskID)
		if err != nil {
			log.Printf("failed to remove cgroup %s: %v", taskID, err)
		}
		return "", basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to create cgroup", err)
	}

	// exec.CommandContext() calls cmd.Process.Kill() on context cancelation which kills just the first process
	// and not the entire process group. We want the whole process group to be killed on context cancelation.
	// So we later call syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) to kill the entire process group in the event
	// of a context cancelation.
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

	startTime := time.Now()
	// Get the process group ID
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// if we fail to get pgid fallback to PID
		pgid = cmd.Process.Pid
	}
	// we can now safely close the CgroupFD
	cgroupFd.Close()

	// Create the new task and add it to the task manager
	task := CreateNewTask(taskID, clientID.(string), pgid, startTime)
	tm.addTask(task)

	// Start monitoring the process
	go tm.monitorProcess(taskID, cmd)

	return taskID, nil
}
