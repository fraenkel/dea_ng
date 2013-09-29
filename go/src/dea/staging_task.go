package dea

type StagingTask struct {
    id string
}

func NewStagingTask(attributes map[string]string) *StagingTask {
    return &StagingTask{id: attributes["task_id"]}
}
