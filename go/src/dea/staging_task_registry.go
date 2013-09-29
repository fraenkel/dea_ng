package dea

type StagingTaskRegistry struct {
    tasks map[string]*StagingTask
}

func NewStagingTaskRegistry() *StagingTaskRegistry {
    return &StagingTaskRegistry{make(map[string]*StagingTask)}
}

func (stagingRegistry *StagingTaskRegistry) Register(task *StagingTask) {
    stagingRegistry.tasks[task.id] = task
}

func (stagingRegistry *StagingTaskRegistry) Unregister(task *StagingTask) {
    delete(stagingRegistry.tasks, task.id)
}

func (stagingRegistry *StagingTaskRegistry) Task(task_id string) *StagingTask {
    return stagingRegistry.tasks[task_id]
}

func (stagingRegistry *StagingTaskRegistry) Each(block func(*StagingTask)) {
    for _, task := range stagingRegistry.tasks {
        block(task)
    }
}
