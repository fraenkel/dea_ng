package responders

import (
	"dea/config"
	"dea/droplet"
	"dea/loggregator"
	"dea/staging"
	"dea/utils"
	"encoding/json"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var stagingLogger = utils.Logger("Staging", nil)

type AppManager interface {
	StartApp(map[string]interface{})
	SaveSnapshot()
}

type Staging struct {
	enabled          bool
	appManager       AppManager
	nats             yagnats.NATSClient
	id               string
	stagingRegistry  *staging.StagingTaskRegistry
	config           *config.Config
	dropletRegistry  *droplet.DropletRegistry
	staging_sid      *int
	staging_id_sid   *int
	staging_stop_sid *int
}

func NewStaging(appManager AppManager, nats yagnats.NATSClient, id string,
	stagingTaskRegistry *staging.StagingTaskRegistry,
	config *config.Config, dropletRegistry *droplet.DropletRegistry) *Staging {
	return &Staging{
		enabled:         config.Staging.Enabled,
		appManager:      appManager,
		nats:            nats,
		id:              id,
		stagingRegistry: stagingTaskRegistry,
		config:          config,
		dropletRegistry: dropletRegistry,
	}
}

func (s *Staging) Start() {
	if !s.enabled {
		return
	}
	staging_sid, err := s.nats.Subscribe("staging", s.handle)
	if err != nil {
		stagingLogger.Error(err.Error())
		return
	}
	s.staging_sid = &staging_sid

	staging_id_sid, err := s.nats.Subscribe("staging."+s.id+".start", s.handle)
	if err != nil {
		stagingLogger.Error(err.Error())
		return
	}
	s.staging_id_sid = &staging_id_sid

	staging_stop_sid, err := s.nats.Subscribe("staging.stop", s.handleStop)
	if err != nil {
		stagingLogger.Error(err.Error())
		return
	}
	s.staging_stop_sid = &staging_stop_sid
}

func (s *Staging) Stop() {
	if s.staging_sid != nil {
		s.nats.Unsubscribe(*s.staging_sid)
	}
	if s.staging_id_sid != nil {
		s.nats.Unsubscribe(*s.staging_id_sid)
	}
	if s.staging_stop_sid != nil {
		s.nats.Unsubscribe(*s.staging_stop_sid)
	}
}

func (s *Staging) handle(msg *yagnats.Message) {
	defer func() {
		if r := recover(); r != nil {
			stagingLogger.Errorf("Error during handle: %v", r)
		}
	}()

	var data map[string]interface{}
	err := json.Unmarshal(msg.Payload, &data)
	if err != nil {
		stagingLogger.Errorf("Parsing failed: %s", err.Error())
		return
	}

	stagingMsg := staging.NewStagingMessage(data)
	appId := stagingMsg.App_id()

	logger := logger_for_app(appId)

	loggregator.Emit(appId, "Got staging request for app with id "+appId)
	logger.Infof("staging.handle.start %v", stagingMsg)
	task := s.stagingRegistry.NewStagingTask(s.config, stagingMsg, s.dropletRegistry, logger)

	s.stagingRegistry.Register(task)

	s.appManager.SaveSnapshot()

	s.notify_setup_completion(msg.ReplyTo, task)
	s.notify_completion(data, task)
	s.notify_upload(msg.ReplyTo, task)
	s.notify_stop(msg.ReplyTo, task)

	task.Start()
}

func (s *Staging) handleStop(msg *yagnats.Message) {
	defer func() {
		if r := recover(); r != nil {
			stagingLogger.Errorf("Error during handleStop: %v", r)
		}
	}()

	var data map[string]interface{}
	err := json.Unmarshal(msg.Payload, &data)
	if err != nil {
		stagingLogger.Errorf("Parsing failed: %s", err.Error())
		return
	}

	appId := data["app_id"].(string)
	if appId == "" {
		stagingLogger.Errorf("Missing app_id: %s", data)
		return
	}

	for _, st := range s.stagingRegistry.Tasks() {

		if appId == st.StagingMessage().App_id() {
			st.Stop()
		}
	}
}

func (s *Staging) notify_setup_completion(replyTo string, task staging.StagingTask) {
	task.SetAfter_setup_callback(func(e error) {
		data := map[string]string{
			"task_id":           task.Id(),
			"streaming_log_url": task.StreamingLogUrl(),
		}
		if e != nil {
			data["error"] = e.Error()

		}
		s.respondTo(replyTo, data)
	})
}

func (s *Staging) notify_completion(data map[string]interface{}, task staging.StagingTask) {
	task.SetAfter_complete_callback(func(e error) {
		if msg, exists := data["start_message"]; exists && e == nil {
			startMsg := msg.(map[string]interface{})
			startMsg["sha1"] = task.DropletSHA1()
			s.appManager.StartApp(startMsg)
		}
	})
}

func (s *Staging) notify_upload(replyTo string, task staging.StagingTask) {
	task.SetAfter_upload_callback(func(e error) {
		data := map[string]string{
			"task_id":            task.Id(),
			"detected_buildpack": task.DetectedBuildpack(),
			"droplet_sha1":       task.DropletSHA1(),
		}
		if e != nil {
			data["error"] = e.Error()
		}

		s.respondTo(replyTo, data)

		s.stagingRegistry.Unregister(task)

		s.appManager.SaveSnapshot()
	})
}

func (s *Staging) notify_stop(replyTo string, task staging.StagingTask) {
	task.SetAfter_stop_callback(func(e error) {
		data := map[string]string{
			"task_id": task.Id(),
		}
		if e != nil {
			data["error"] = e.Error()
		}

		s.respondTo(replyTo, data)

		s.stagingRegistry.Unregister(task)

		s.appManager.SaveSnapshot()
	})
}

func (s *Staging) respondTo(replyTo string, params map[string]string) {
	data := map[string]*string{
		"task_id":                stringOrNil(params["task_id"]),
		"task_streaming_log_url": stringOrNil(params["streaming_log_url"]),
		"detected_buildpack":     stringOrNil(params["detected_buildpack"]),
		"error":                  stringOrNil(params["error"]),
		"droplet_sha1":           stringOrNil(params["droplet_sha1"]),
	}

	if bytes, err := json.Marshal(&data); err != nil {
		stagingLogger.Errorf("Marshal failed with %v", data)
	} else {
		s.nats.Publish(replyTo, bytes)
	}
}

func stringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func logger_for_app(app_id string) *steno.Logger {
	return utils.Logger("Staging", map[string]interface{}{"app_guid": app_id})
}
