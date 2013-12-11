package staging

import (
	"dea"
	cfg "dea/config"
	steno "github.com/cloudfoundry/gosteno"
	"sync"
)

type StagingTaskCreator func(*cfg.Config, dea.StagingMessage, []dea.StagingBuildpack, dea.DropletRegistry, *steno.Logger) dea.StagingTask

type StagingTaskRegistry struct {
	sync.Mutex
	tasks           map[string]dea.StagingTask
	taskCreator     StagingTaskCreator
	dropletRegistry dea.DropletRegistry
	config          *cfg.Config
}

func NewStagingTaskRegistry(config *cfg.Config, dregistry dea.DropletRegistry, creator StagingTaskCreator) *StagingTaskRegistry {
	return &StagingTaskRegistry{
		tasks:           make(map[string]dea.StagingTask),
		taskCreator:     creator,
		dropletRegistry: dregistry,
		config:          config,
	}
}

func (s *StagingTaskRegistry) NewStagingTask(staging_message dea.StagingMessage, logger *steno.Logger) dea.StagingTask {
	buildpacks := s.BuildpacksInUse()
	return s.taskCreator(s.config, staging_message, buildpacks, s.dropletRegistry, logger)
}

func (s *StagingTaskRegistry) Register(task dea.StagingTask) {
	s.Lock()
	defer s.Unlock()

	s.tasks[task.Id()] = task
}

func (s *StagingTaskRegistry) Unregister(task dea.StagingTask) {
	s.Lock()
	defer s.Unlock()

	delete(s.tasks, task.Id())
}

func (s *StagingTaskRegistry) Task(task_id string) dea.StagingTask {
	s.Lock()
	defer s.Unlock()

	return s.tasks[task_id]
}

func (s *StagingTaskRegistry) Tasks() []dea.StagingTask {
	s.Lock()
	defer s.Unlock()

	tasks := make([]dea.StagingTask, 0, len(s.tasks))
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

func (s *StagingTaskRegistry) BuildpacksInUse() []dea.StagingBuildpack {
	inuse := make(map[string]bool)
	buildpacks := make([]dea.StagingBuildpack, 0, 5)

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
