package starting

import (
	"dea"
	"strconv"
)

const (
	HOST = "0.0.0.0"
)

type RunningEnv struct {
	instance dea.Instance
}

func NewRunningEnv(instance dea.Instance) *RunningEnv {
	return &RunningEnv{
		instance: instance,
	}
}

func (r RunningEnv) StartMessage() dea.StartMessage {
	return r.instance.StartMessage()
}

func (r RunningEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 5)
	vars[0] = []string{"HOME", "$PWD/app"}
	vars[1] = []string{"TMPDIR", "$PWD/tmp"}
	vars[2] = []string{"VCAP_APP_HOST", HOST}
	vars[3] = []string{"VCAP_APP_PORT", strconv.FormatUint(uint64(r.instance.ContainerPort()), 10)}
	vars[4] = []string{"PORT", "$VCAP_APP_PORT"}
	return vars
}

func (r RunningEnv) VcapApplication() map[string]interface{} {
	started_at := r.instance.StateTime(dea.STATE_STARTING)
	return map[string]interface{}{
		"instance_id":          r.instance.Id(),
		"instance_index":       r.instance.Index(),
		"host":                 HOST,
		"port":                 r.instance.ContainerPort(),
		"started_at":           started_at,
		"started_at_timestamp": started_at.Unix(),
		"start":                started_at,
		"state_timestamp":      started_at.Unix(),
	}
}
