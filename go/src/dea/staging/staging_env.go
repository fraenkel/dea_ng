package staging

import (
	"dea"
	"strconv"
	"time"
)

type StagingEnv struct {
	stagingTask dea.StagingTask
}

func NewStagingEnv(stagingTask dea.StagingTask) *StagingEnv {
	return &StagingEnv{
		stagingTask: stagingTask,
	}
}

func (s StagingEnv) StartMessage() dea.StartMessage {
	return s.stagingTask.StagingMessage().StartMessage()
}

func (s StagingEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 3)

	buildpackCache := s.stagingTask.StagingConfig().Environment["BUILDPACK_CACHE"]

	vars[0] = []string{"BUILDPACK_CACHE", buildpackCache}
	vars[1] = []string{"STAGING_TIMEOUT", strconv.FormatUint(uint64(s.stagingTask.StagingTimeout()/time.Second), 10)}
	vars[2] = []string{"MEMORY_LIMIT", strconv.FormatUint(s.StartMessage().MemoryLimitMB(), 10) + "m"}
	return vars
}

func (s StagingEnv) VcapApplication() map[string]interface{} {
	return map[string]interface{}{}
}
