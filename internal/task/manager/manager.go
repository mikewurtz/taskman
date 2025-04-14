package task

import (
	"context"
	"fmt"
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

func NewTaskManager(ctx context.Context) *TaskManager {
	return &TaskManager{
		tasksMapByID: make(map[string]*Task),
		ctx:          ctx,
	}
}

func (tm *TaskManager) addTask(task *Task) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task.done = make(chan struct{})
	tm.tasksMapByID[task.GetID()] = task
}

// WaitForTasks waits for all tasks to complete with a timeout
func (tm *TaskManager) WaitForTasks() error {
	tm.mu.RLock()
	tasks := make([]*Task, 0, len(tm.tasksMapByID))
	for _, task := range tm.tasksMapByID {
		tasks = append(tasks, task)
	}
	tm.mu.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(tasks))

	// send done to all the tasks
	for _, task := range tasks {
		go func(t *Task) {
			<-t.done
			wg.Done()
		}(task)
	}

	// Create a done channel that signals when all tasks are complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// wait until all the taks are done or we timeout
	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for tasks to complete")
	}
}

func (tm *TaskManager) getTaskFromMap(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasksMapByID[taskID]
	if !ok {
		return nil, basetask.NewTaskError(basetask.ErrNotFound, "task with id %s not found", taskID)
	}
	return task, nil
}
