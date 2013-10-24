package staging

import (
	"dea/config"
	"sync"
)

type StagingTaskRegistry struct {
	sync.Mutex
	tasks map[string]*StagingTask
}

func NewStagingTaskRegistry() *StagingTaskRegistry {
	return &StagingTaskRegistry{
		tasks: make(map[string]*StagingTask),
	}
}

func (s StagingTaskRegistry) Register(task *StagingTask) {
	s.Lock()
	defer s.Unlock()

	s.tasks[task.id] = task
}

func (s StagingTaskRegistry) Unregister(task *StagingTask) {
	s.Lock()
	defer s.Unlock()

	delete(s.tasks, task.id)
}

func (s StagingTaskRegistry) Task(task_id string) *StagingTask {
	s.Lock()
	defer s.Unlock()

	return s.tasks[task_id]
}

func (s StagingTaskRegistry) Tasks() []*StagingTask {
	s.Lock()
	defer s.Unlock()

	tasks := make([]*StagingTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (s StagingTaskRegistry) ReservedMemory() config.Memory {
	var sum config.Memory = 0
	for _, t := range s.Tasks() {
		sum = sum + t.MemoryLimit()
	}
	return sum
}

func (s StagingTaskRegistry) ReservedDisk() config.Disk {
	var sum config.Disk = 0
	for _, t := range s.Tasks() {
		sum = sum + t.DiskLimit()
	}
	return sum
}
