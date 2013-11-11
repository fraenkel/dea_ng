package starting

import (
	"dea/env"
	"strconv"
)

const (
	HOST = "0.0.0.0"
)

type RunningEnv struct {
	instance *Instance
}

func NewRunningEnv(instance *Instance) *RunningEnv {
	return &RunningEnv{
		instance: instance,
	}
}

func (r RunningEnv) Message() env.Message {
	return &r.instance.startData
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
	started_at := r.instance.StateTime(STATE_STARTING)
	return map[string]interface{}{
		"instance_id":          r.instance.Id(),
		"instance_index":       r.Message().Index(),
		"host":                 HOST,
		"port":                 strconv.FormatUint(uint64(r.instance.ContainerPort()), 10),
		"started_at":           started_at.String(),
		"started_at_timestamp": started_at.Unix(),
		"start":                started_at.String(),
		"state_timestamp":      started_at.Unix(),
	}
}
