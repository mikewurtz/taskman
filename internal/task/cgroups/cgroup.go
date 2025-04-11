package cgroups

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const baseCgroupPath = "/sys/fs/cgroup/"

// CreateCgroupForTask creates a cgroup for a task
// In a real system, we would want the cgroup config to be configurable
func CreateCgroupForTask(taskID string) (*os.File, error) {
	cgroupPath := filepath.Join(baseCgroupPath, taskID)

	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cgroup directory %s: %w", cgroupPath, err)
	}

	// Configure CPU limits: Quota 200000 µs and Period 1000000 µs.
	// this should cap the cpu usage to 20%
	cpuMaxPath := filepath.Join(cgroupPath, "cpu.max")
	cpuConfig := "200000 1000000"
	if err := os.WriteFile(cpuMaxPath, []byte(cpuConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write CPU config to %s: %w", cpuMaxPath, err)
	}

	// Configure memory limit: 64M.
	memoryMaxPath := filepath.Join(cgroupPath, "memory.max")
	memoryConfig := "64M"
	if err := os.WriteFile(memoryMaxPath, []byte(memoryConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write memory config to %s: %w", memoryMaxPath, err)
	}

	// io is not always enabled on the system and can be enabled by:
	// echo "+io" | sudo tee /sys/fs/cgroup/cgroup.subtree_control
	// Configure IO limits: device "8:0" with max read and write bandwidth 1048576 (1 MB/s).
	ioMaxPath := filepath.Join(cgroupPath, "io.max")
	ioConfig := "8:0 rbps=1048576 wbps=1048576"
	if err := os.WriteFile(ioMaxPath, []byte(ioConfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write IO config to %s: %w", ioMaxPath, err)
	}

	// Open the cgroup directory as a file descriptor
	cgFd, err := os.Open(cgroupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cgroup path: %w", err)
	}

	return cgFd, nil
}

// RemoveCgroupForTask removes the cgroup for a task
func RemoveCgroupForTask(taskID string) error {
	cgroupPath := filepath.Join(baseCgroupPath, taskID)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("failed to remove cgroup directory %s: timeout reached", cgroupPath)
		case <-ticker.C:
			// attempt to remove the cgroup directory
			if err := os.Remove(cgroupPath); err != nil {
				// If the error is due to the directory not being empty or busy, continue waiting
				if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EBUSY) {
					continue
				}
				// For any other error, return immediately
				return fmt.Errorf("failed to remove cgroup directory %s: %w", cgroupPath, err)
			}
			// Successfully removed the directory
			return nil
		}
	}
}

// CheckIfOOMKilled checks if the task has been OOM killed
func CheckIfOOMKilled(taskID string) (bool, error) {
	cgroupPath := filepath.Join(baseCgroupPath, taskID)
	oomPath := filepath.Join(cgroupPath, "memory.events")

	data, err := os.ReadFile(oomPath)
	if err != nil {
		return false, fmt.Errorf("failed to read memory.events: %w", err)
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "oom_kill" && fields[1] == "1" {
			return true, nil
		}
	}

	return false, nil
}
