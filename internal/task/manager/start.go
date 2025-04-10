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
	clientCN := ctx.Value(basegrpc.ClientIDKey)
	log.Printf("Starting task for client %s: %s %v", clientCN, command, args)

	if command == "" {
		return "", basetask.NewTaskError(basetask.ErrInvalidArgument, "command cannot be empty")
	}

	cmd := exec.CommandContext(ctx, command, args...)
	taskID := uuid.New().String()

	// Create cgroup and get file descriptor
	cgroupFd, err := cgroups.CreateCgroupForTask(taskID)
	if err != nil {
		return "", basetask.NewTaskErrorWithErr(basetask.ErrInternal, "failed to create cgroup", err)
	}

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

	// we can now safely close the CgroupFD
	cgroupFd.Close()

	// Get the process group ID
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// if we fail to get pgid fallback to PID
		pgid = cmd.Process.Pid
	}


	task := &Task{
		ID:        taskID,
		StartTime: time.Now(),
		ClientID:  clientCN.(string),
		Status:    "JOB_STATUS_STARTED",
		ProcessID: pgid,
	}

	tm.AddTask(task)

	// Start monitoring the process
	go tm.monitorProcess(taskID, cmd)

	return taskID, nil
}
