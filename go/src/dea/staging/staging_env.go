package staging

import (
	"dea/env"
	"strconv"
)

type StagingEnv struct {
	stagingTask *StagingTask
}

func NewStagingEnv(stagingTask *StagingTask) *StagingEnv {
	return &StagingEnv{
		stagingTask: stagingTask,
	}
}

func (s StagingEnv) Message() env.Message {
	return s.stagingTask.StagingMessage().start_data()
}

func (s StagingEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 3)

	buildpackCache := s.stagingTask.stagingConfig.Environment["BUILDPACK_CACHE"]

	vars[0] = []string{"BUILDPACK_CACHE", buildpackCache}
	vars[1] = []string{"STAGING_TIMEOUT", strconv.FormatUint(uint64(s.stagingTask.StagingTimeout()), 10)}
	vars[2] = []string{"MEMORY_LIMIT", strconv.FormatUint(s.Message().MemoryLimit(), 10) + "m"}
	return vars
}

func (s StagingEnv) VcapApplication() map[string]interface{} {
	return map[string]interface{}{}
}
