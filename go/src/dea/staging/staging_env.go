package staging

import (
	"strconv"
)

type StagingEnv struct {
	data        map[string]interface{}
	stagingTask *StagingTask
}

func NewStagingEnv(data map[string]interface{}, stagingTask *StagingTask) *StagingEnv {
	startMessage := data["start_message"].(map[string]interface{})
	return &StagingEnv{
		data:        startMessage,
		stagingTask: stagingTask,
	}
}

func (s StagingEnv) Data() map[string]interface{} {
	return s.data
}

func (s StagingEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 0, 4)

	buildpackCache := s.stagingTask.stagingConfig.Environment["BUILDPACK_CACHE"]

	memoryLimit := ""
	if limitsConfig, exists := s.data["limits"]; exists {
		if limits, ok := limitsConfig.(map[string]interface{}); ok {
			if memory, ok := limits["mem"].(uint64); ok {
				memoryLimit = strconv.FormatUint(memory, 10)
			}
		}
	}

	vars[1] = []string{"BUILDPACK_CACHE", buildpackCache}
	vars[2] = []string{"STAGING_TIMEOUT", strconv.FormatUint(uint64(s.stagingTask.StagingTimeout()), 10)}
	vars[3] = []string{"MEMORY_LIMIT", memoryLimit + "m"}
	return vars
}

func (s StagingEnv) VcapApplication() map[string]interface{} {
	return map[string]interface{}{}
}
