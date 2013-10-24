package starting

import (
	"strconv"
)

const (
	HOST = "0.0.0.0"
)

type RunningEnv struct {
	data     map[string]interface{}
	instance Instance
}

func NewRunningEnv(data map[string]interface{}, instance Instance) *RunningEnv {
	startMessage := data["start_message"].(map[string]interface{})
	return &RunningEnv{
		data:     startMessage,
		instance: instance,
	}
}

func (r RunningEnv) Data() map[string]interface{} {
	return r.data
}

func (r RunningEnv) ExportedSystemEnvironmentVariables() [][]string {
	vars := make([][]string, 0, 5)
	vars[0] = []string{"HOME", "$PWD/app"}
	vars[1] = []string{"TMPDIR", "$PWD/tmp"}
	vars[2] = []string{"VCAP_APP_HOST", HOST}
	vars[3] = []string{"VCAP_APP_PORT", strconv.FormatUint(uint64(r.instance.ContainerPort()), 10)}
	vars[4] = []string{"PORT", "$VCAP_APP_PORT"}
	return vars
}

func (r RunningEnv) VcapApplication() map[string]interface{} {
	started_at := r.instance.StateTimestamp(STATE_RUNNING)
	return map[string]interface{}{
		"instance_id":          r.instance.Id(),
		"instance_index":       r.instance.Index(),
		"host":                 HOST,
		"port":                 strconv.FormatUint(uint64(r.instance.ContainerPort()), 10),
		"started_at":           started_at.String(),
		"started_at_timestamp": started_at.Unix(),
		"start":                started_at.String(),
		"state_timestamp":      started_at.Unix(),
	}
}
