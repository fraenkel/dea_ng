package responders

import (
	"dea/config"
	"dea/droplet"
	"dea/loggregator"
	"dea/staging"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/go_cfmessagebus"
	"reflect"
)

type AppStarter interface {
	StartApp(map[string]interface{})
}

type Staging struct {
	enabled         bool
	appStarter      AppStarter
	mbus            cfmessagebus.MessageBus
	id              string
	stagingRegistry *staging.StagingTaskRegistry
	config          *config.Config
	dropletRegistry *droplet.DropletRegistry
}

func NewStaging(starter AppStarter, mbus cfmessagebus.MessageBus, id string,
	stagingTaskRegistry *staging.StagingTaskRegistry,
	config *config.Config, dropletRegistry *droplet.DropletRegistry) *Staging {
	return &Staging{
		enabled:         config.Staging.Enabled,
		appStarter:      starter,
		mbus:            mbus,
		id:              id,
		stagingRegistry: stagingTaskRegistry,
		config:          config,
		dropletRegistry: dropletRegistry,
	}
}

func (s Staging) Start() {
	if !s.enabled {
		return
	}
	if err := s.mbus.ReplyToChannel("staging", s.handle); err != nil {
		utils.Logger("Staging").Error(err.Error())
		return
	}

	if err := s.mbus.ReplyToChannel("staging."+s.id+".start", s.handle); err != nil {
		utils.Logger("Staging").Error(err.Error())
		return
	}

	if err := s.mbus.ReplyToChannel("staging.stop", s.handle); err != nil {
		utils.Logger("Staging").Error(err.Error())
		return
	}
}

func (s Staging) Stop() {
	//	unsubscribe_from_staging
	//	unsubscribe_from_dea_specific_staging
	//	unsubscribe_from_staging_stop
}

func (s Staging) handle(payload []byte, reply cfmessagebus.ReplyTo) {

	var tmpVal interface{}
	err := json.Unmarshal(payload, &tmpVal)
	if err != nil {
		utils.Logger("Staging").Errorf("Parsing failed: %s", err.Error)
		return
	}

	data := tmpVal.(map[string]interface{})
	appId := data["app_id"].(string)

	loggregator.Emit(appId, "Got staging request for app with id "+appId)
	utils.Logger("Staging").Infof("Got staging request with %v", data)
	task := staging.NewStagingTask(s.config, data, buildpacksInUse(s.stagingRegistry), s.dropletRegistry)

	s.stagingRegistry.Register(&task)

	s.notify_setup_completion(reply, &task)
	s.notify_completion(data, reply, &task)
	s.notify_upload(reply, &task)
	s.notify_stop(reply, &task)

	task.Start()
}

func (s Staging) notify_setup_completion(reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.SetAfter_setup_callback(func(e error) {
		data := map[string]string{
			"task_id":           task.Id(),
			"streaming_log_url": task.StreamingLogUrl(),
		}
		if e != nil {
			data["error"] = e.Error()

		}
		respondTo(reply, data)
	})
}

func (s Staging) notify_completion(data map[string]interface{}, reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.SetAfter_complete_callback(func(e error) {
		if msg, exists := data["start_message"]; exists && e != nil {
			startMsg := msg.(map[string]interface{})
			startMsg["sha1"] = task.DropletSHA1()
			s.appStarter.StartApp(startMsg)
		}
	})
}

func (s Staging) notify_upload(reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.SetAfter_upload_callback(func(e error) {
		data := map[string]string{
			"task_id":            task.Id(),
			"detected_buildpack": task.DetectedBuildpack(),
			"droplet_sha1":       task.DropletSHA1(),
		}
		if e != nil {
			data["error"] = e.Error()
		}

		respondTo(reply, data)

		s.stagingRegistry.Unregister(task)
	})
}

func (s Staging) notify_stop(reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.SetAfter_stop_callback(func(e error) {
		data := map[string]string{
			"task_id": task.Id(),
		}
		if e != nil {
			data["error"] = e.Error()
		}

		respondTo(reply, data)

		s.stagingRegistry.Unregister(task)
	})
}

func respondTo(reply cfmessagebus.ReplyTo, params map[string]string) {
	data := map[string]string{
		"task_id":                params["task_id"],
		"task_streaming_log_url": params["streaming_log_url"],
		"detected_buildpack":     params["detected_buildpack"],
		"error":                  params["error"],
		"droplet_sha1":           params["droplet_sha1"],
	}

	if bytes, err := json.Marshal(&data); err != nil {
		utils.Logger("Staging").Errorf("Marshal failed with %v", data)
	} else {
		reply.Respond(bytes)
	}
}

func buildpacksInUse(stagingRegistry *staging.StagingTaskRegistry) []map[string]string {
	buildpacks := make([]map[string]string, 0, 10)
	for _, t := range stagingRegistry.Tasks() {
		for _, bp := range t.AdminBuildpacks() {
			if !buildpack_included(buildpacks, bp) {
				buildpacks = append(buildpacks, bp)
			}
		}
	}

	return buildpacks
}

func buildpack_included(buildpacks []map[string]string, item map[string]string) bool {
	for _, v := range buildpacks {
		if reflect.DeepEqual(v, item) {
			return true
		}
	}
	return false
}
