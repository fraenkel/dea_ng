package health_check

import (
	"dea/utils"
	"time"
)

type StateFileReady struct {
	path     string
	retry    time.Duration
	deferred HealthCheckCallback
	end_time time.Time
	done     bool
}

func NewStateFileReady(path string, retryInterval time.Duration, hc HealthCheckCallback, timeout time.Duration) StateFileReady {
	sfr := StateFileReady{
		path:     path,
		retry:    retryInterval,
		deferred: hc,
		end_time: time.Now().Add(timeout),
	}

	go sfr.check_state_file()
	return sfr
}

func (s StateFileReady) check_state_file() {
	if s.done {
		return
	}

	if utils.File_Exists(s.path) {
		var state map[string]interface{}
		if err := utils.Yaml_Load(s.path, state); err == nil {
			if state["state"] == "RUNNING" {
				s.done = true
				s.deferred.Success()
				return
			}
		} else {
			utils.Logger("StateFileReady").Errorf("Failed parsing state file: ", err.Error())
		}
	}

	if time.Now().Before(s.end_time) {
		time.AfterFunc(s.retry, s.check_state_file)
	} else {
		s.done = true
		s.deferred.Failure()
	}
}

func (s StateFileReady) Destroy() {
}
