package task

import (
	"sync"
	"time"
)

type TaskManager struct {
	mu sync.RWMutex
	// task map by task ID
	tasksMapByID map[string]*Task
}

type Task struct {
	mu                sync.RWMutex
	ID                string
	ClientID          string
	ProcessID         int
	Status            string
	StartTime         time.Time
	ExitCode          *int32
	TerminationSignal string
	TerminationSource string
	EndTime           time.Time
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasksMapByID: make(map[string]*Task),
	}
}

func (tm *TaskManager) AddTask(task *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tasksMapByID[task.ID] = task
}
