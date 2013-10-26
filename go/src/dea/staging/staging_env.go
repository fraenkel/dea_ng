package staging

import (
	"dea/env"
	"strconv"
)

type StagingEnv struct {
	startMsg    env.Message
	stagingTask *StagingTask
}

func NewStagingEnv(stagingMsg StagingMessage, stagingTask *StagingTask) *StagingEnv {
	return &StagingEnv{
		startMsg:    stagingMsg.start_message(),
		stagingTask: stagingTask,
	}
}

func (s StagingEnv) Message() env.Message {
	return s.startMsg
}

func (s StagingEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 0, 4)

	buildpackCache := s.stagingTask.stagingConfig.Environment["BUILDPACK_CACHE"]

	vars[1] = []string{"BUILDPACK_CACHE", buildpackCache}
	vars[2] = []string{"STAGING_TIMEOUT", strconv.FormatUint(uint64(s.stagingTask.StagingTimeout()), 10)}
	vars[3] = []string{"MEMORY_LIMIT", strconv.FormatUint(s.startMsg.MemoryLimit(), 10) + "m"}
	return vars
}

func (s StagingEnv) VcapApplication() map[string]interface{} {
	return map[string]interface{}{}
}
