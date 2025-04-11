package task

import (
	"context"
	"sync"
	"time"

	basetask "github.com/mikewurtz/taskman/internal/task"
)

type TaskManager struct {
	mu sync.RWMutex
	// task map by task ID
	tasksMapByID map[string]*Task
	ctx          context.Context
}

type Task struct {
	mu                sync.RWMutex
	ID                string
	ClientID          string
	ProcessID         int
	Status            int
	StartTime         time.Time
	ExitCode          *int32
	TerminationSignal string
	TerminationSource string
	EndTime           time.Time
	done              chan struct{}
	doOnce            sync.Once
}

func NewTaskManager(ctx context.Context) *TaskManager {
	return &TaskManager{
		tasksMapByID: make(map[string]*Task),
		ctx:          ctx,
	}
}

func (tm *TaskManager) AddTask(task *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task.done = make(chan struct{})
	tm.tasksMapByID[task.ID] = task
}

func (tm *TaskManager) WaitForTasks() {
	tm.mu.RLock()
	tasks := make([]*Task, 0, len(tm.tasksMapByID))
	for _, task := range tm.tasksMapByID {
		tasks = append(tasks, task)
	}
	tm.mu.RUnlock()

	for _, task := range tasks {
		<-task.done
	}
}

func (tm *TaskManager) GetTask(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasksMapByID[taskID]
	if !ok {
		return nil, basetask.NewTaskError(basetask.ErrNotFound, "task with id %s not found", taskID)
	}
	return task, nil
}
