package staging

import (
	cfg "dea/config"
	"dea/droplet"
	steno "github.com/cloudfoundry/gosteno"
	"sync"
)

type StagingTaskCreator func(*cfg.Config, StagingMessage, []StagingBuildpack, droplet.DropletRegistry, *steno.Logger) StagingTask

type StagingTaskRegistry struct {
	sync.Mutex
	tasks       map[string]StagingTask
	taskCreator StagingTaskCreator
}

func NewStagingTaskRegistry(creator StagingTaskCreator) *StagingTaskRegistry {
	return &StagingTaskRegistry{
		tasks:       make(map[string]StagingTask),
		taskCreator: creator,
	}
}

func (s *StagingTaskRegistry) NewStagingTask(config *cfg.Config, staging_message StagingMessage,
	dropletRegistry droplet.DropletRegistry, logger *steno.Logger) StagingTask {
	buildpacks := s.BuildpacksInUse()
	return s.taskCreator(config, staging_message, buildpacks, dropletRegistry, logger)
}

func (s *StagingTaskRegistry) Register(task StagingTask) {
	s.Lock()
	defer s.Unlock()

	s.tasks[task.Id()] = task
}

func (s *StagingTaskRegistry) Unregister(task StagingTask) {
	s.Lock()
	defer s.Unlock()

	delete(s.tasks, task.Id())
}

func (s *StagingTaskRegistry) Task(task_id string) StagingTask {
	s.Lock()
	defer s.Unlock()

	return s.tasks[task_id]
}

func (s *StagingTaskRegistry) Tasks() []StagingTask {
	s.Lock()
	defer s.Unlock()

	tasks := make([]StagingTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *StagingTaskRegistry) ReservedMemory() cfg.Memory {
	var sum cfg.Memory = 0
	for _, t := range s.Tasks() {
		sum = sum + t.MemoryLimit()
	}
	return sum
}

func (s *StagingTaskRegistry) ReservedDisk() cfg.Disk {
	var sum cfg.Disk = 0
	for _, t := range s.Tasks() {
		sum = sum + t.DiskLimit()
	}
	return sum
}

func (s *StagingTaskRegistry) BuildpacksInUse() []StagingBuildpack {
	inuse := make(map[string]bool)
	buildpacks := make([]StagingBuildpack, 0, 5)

	for _, t := range s.Tasks() {
		for _, bp := range t.StagingMessage().AdminBuildpacks() {
			key := bp.Key + "!" + bp.Url.String()
			if inuse[key] == false {
				inuse[key] = true
				buildpacks = append(buildpacks, bp)
			}
		}
	}

	return buildpacks
}
