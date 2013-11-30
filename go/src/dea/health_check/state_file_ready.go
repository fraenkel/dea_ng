package health_check

import (
	"dea/utils"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

type StateFileReady struct {
	path     string
	retry    time.Duration
	deferred HealthCheckCallback
	End_time time.Time
	done     bool
}

func NewStateFileReady(path string, retryInterval time.Duration, hc HealthCheckCallback, timeout time.Duration) StateFileReady {
	sfr := StateFileReady{
		path:     path,
		retry:    retryInterval,
		deferred: hc,
		End_time: time.Now().Add(timeout),
	}

	go sfr.check_state_file()
	return sfr
}

func (s StateFileReady) check_state_file() {
	if s.done {
		return
	}

	if utils.File_Exists(s.path) {
		file, err := os.Open(s.path)
		if err == nil {
			bytes, err := ioutil.ReadAll(file)
			file.Close()

			if err == nil {
				var state = make(map[string]interface{})
				err = json.Unmarshal(bytes, &state)
				if err == nil {
					switch state["state"] {
					case "RUNNING":
						s.done = true
						s.deferred.Success()
					case "CRASHED":
						s.done = true
						s.deferred.Failure()
					}
					if s.done {
						return
					}
				}
			}
		}

		if err != nil {
			utils.Logger("StateFileReady", nil).Errorf("Failed parsing state file: ", err.Error())
		}
	}

	if time.Now().Before(s.End_time) {
		time.AfterFunc(s.retry, s.check_state_file)
	} else {
		s.done = true
		s.deferred.Failure()
	}
}

func (s StateFileReady) Destroy() {
}
