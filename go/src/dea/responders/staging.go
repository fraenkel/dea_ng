package responders

import (
	"dea"
	"dea/config"
	"dea/loggregator"
	"dea/staging"
	"dea/utils"
	"encoding/json"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

var stagingLogger = utils.Logger("Staging", nil)

type Staging struct {
	enabled          bool
	instanceManager  dea.InstanceManager
	snapshot         dea.Snapshot
	nats             yagnats.NATSClient
	id               string
	stagingRegistry  dea.StagingTaskRegistry
	config           *config.Config
	dropletRegistry  dea.DropletRegistry
	resourceMgr      dea.ResourceManager
	urlMaker         dea.StagingTaskUrlMaker
	staging_sid      *int
	staging_id_sid   *int
	staging_stop_sid *int
}

func NewStaging(boot dea.Bootstrap, id string, maker dea.StagingTaskUrlMaker) *Staging {
	config := boot.Config()
	return &Staging{
		enabled:         config.Staging.Enabled,
		instanceManager: boot.InstanceManager(),
		snapshot:        boot.Snapshot(),
		nats:            boot.Nats(),
		id:              id,
		stagingRegistry: boot.StagingTaskRegistry(),
		config:          config,
		dropletRegistry: boot.DropletRegistry(),
		resourceMgr:     boot.ResourceManager(),
		urlMaker:        maker,
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

	task := s.stagingRegistry.NewStagingTask(stagingMsg, logger)

	if constrained := s.resourceMgr.GetConstrainedResource(task.MemoryLimit(), task.DiskLimit()); constrained != "" {
		s.respondTo(msg.ReplyTo, map[string]string{
			"task_id": task.Id(),
			"error":   "Not enough " + constrained + " resources available",
		})

		logger.Errord(map[string]interface{}{
			"app_id":                appId,
			"contstrained_resource": constrained,
		}, "staging.start.insufficient-resource")

		return
	}

	s.stagingRegistry.Register(task)

	s.snapshot.Save()

	s.notify_setup_completion(msg.ReplyTo, task)
	s.notify_completion(msg.ReplyTo, data, task)
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
			st.Stop(nil)
		}
	}
}

func (s *Staging) notify_setup_completion(replyTo string, task dea.StagingTask) {
	task.SetAfter_setup_callback(func(e error) error {
		data := map[string]string{
			"task_id":           task.Id(),
			"streaming_log_url": task.StreamingLogUrl(s.urlMaker),
		}
		if e != nil {
			data["error"] = e.Error()

		}
		s.respondTo(replyTo, data)

		return nil
	})
}

func (s *Staging) notify_completion(replyTo string, data map[string]interface{}, task dea.StagingTask) {
	task.SetAfter_complete_callback(func(e error) error {
		response := map[string]string{
			"task_id":            task.Id(),
			"detected_buildpack": task.DetectedBuildpack(),
			"droplet_sha1":       task.DropletSHA1(),
		}

		if e != nil {
			response["error"] = e.Error()
		}

		s.respondTo(replyTo, response)

		// Unregistering the staging task will release the reservation of excess memory reserved for the app,
		// if the app requires more memory than the staging process.
		s.stagingRegistry.Unregister(task)

		s.snapshot.Save()

		if msg, exists := data["start_message"]; exists && e == nil {
			startMsg := msg.(map[string]interface{})
			startMsg["sha1"] = task.DropletSHA1()

			// Now re-reserve the app's memory.  There may be a window between staging task unregistration and here
			// where the DEA could no longer have enough memory to start the app.  In that case, the health manager
			// should cause the app to be relocated on another DEA.
			s.instanceManager.StartApp(startMsg)
		}

		return nil
	})
}

func (s *Staging) notify_stop(replyTo string, task dea.StagingTask) {
	task.SetAfter_stop_callback(func(e error) error {
		data := map[string]string{
			"task_id": task.Id(),
		}
		if e != nil {
			data["error"] = e.Error()
		}

		s.respondTo(replyTo, data)

		s.stagingRegistry.Unregister(task)

		s.snapshot.Save()

		return nil
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
