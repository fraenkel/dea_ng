package responders

import (
	"dea/config"
	"dea/staging"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/loggregatorlib/emitter"
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
	emitter         emitter.Emitter
	config          *config.Config
}

func NewStaging(starter AppStarter, mbus cfmessagebus.MessageBus, id string, stagingTaskRegistry *staging.StagingTaskRegistry, emitter emitter.Emitter, config *config.Config) *Staging {
	return &Staging{
		enabled:         config.Staging.Enabled,
		appStarter:      starter,
		mbus:            mbus,
		id:              id,
		stagingRegistry: stagingTaskRegistry,
		emitter:         emitter,
		config:          config,
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

	s.emitter.Emit(appId, "Got staging request for app with id "+appId)
	utils.Logger("Staging").Infof("Got staging request with %v", data)
	task := staging.NewStagingTask(s.config, data, buildpacksInUse(s.stagingRegistry))

	s.stagingRegistry.Register(&task)

	s.notify_setup_completion(reply, &task)
	s.notify_completion(data, reply, &task)
	s.notify_upload(reply, &task)
	s.notify_stop(reply, &task)

	task.Start(nil)
}

func (s Staging) notify_setup_completion(reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.After_setup_callback(func(e error) {
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
	task.After_complete_callback(func(e error) {
		if msg, exists := data["start_message"]; exists && e != nil {
			startMsg := msg.(map[string]interface{})
			startMsg["sha1"] = task.DropletSHA1()
			s.appStarter.StartApp(startMsg)
		}
	})
}

func (s Staging) notify_upload(reply cfmessagebus.ReplyTo, task *staging.StagingTask) {
	task.After_upload_callback(func(e error) {
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
	task.After_stop_callback(func(e error) {
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

func buildpacksInUse(stagingRegistry *staging.StagingTaskRegistry) []string {
	buildpacks := make(map[string]string)
	for _, t := range stagingRegistry.Tasks() {
		for _, bp := range t.AdminBuildpacks() {
			buildpacks[bp] = bp
		}
	}
	bpList := make([]string, 0, len(buildpacks))
	for k, _ := range buildpacks {
		bpList = append(bpList, k)
	}
	return bpList
}

/*
require "dea/staging_task"
require "dea/loggregator"

module Dea::Responders
  class Staging
    attr_reader :nats
    attr_reader :dea_id
    attr_reader :bootstrap
    attr_reader :staging_task_registry
    attr_reader :dir_server
    attr_reader :config

    def initialize(nats, dea_id, bootstrap, staging_task_registry, dir_server, config)
      @nats = nats
      @dea_id = dea_id
      @bootstrap = bootstrap
      @staging_task_registry = staging_task_registry
      @dir_server = dir_server
      @config = config
    end

    def start
      return unless configured_to_stage?
      subscribe_to_staging
      subscribe_to_dea_specific_staging
      subscribe_to_staging_stop
    end

    def stop
      unsubscribe_from_staging
      unsubscribe_from_dea_specific_staging
      unsubscribe_from_staging_stop
    end

    def handle(message)
      app_id = message.data["app_id"]
      logger = logger_for_app(app_id)
      Dea::Loggregator.emit(app_id, "Got staging request for app with id #{app_id}")
      logger.info("Got staging request with #{message.data.inspect}")

      task = Dea::StagingTask.new(bootstrap, dir_server, message.data, buildpacks_in_use, logger)
      staging_task_registry.register(task)

      notify_setup_completion(message, task)
      notify_completion(message, task)
      notify_upload(message, task)
      notify_stop(message, task)

      task.start
    rescue => e
      logger.error "staging.handle.failed", :error => e, :backtrace => e.backtrace
    end

    def handle_stop(message)
      staging_task_registry.each do |task|
        if message.data["app_id"] == task.attributes["app_id"]
          task.stop
        end
      end
    rescue => e
      logger.error "staging.handle_stop.failed", :error => e, :backtrace => e.backtrace
    end

    private

    def configured_to_stage?
      config["staging"] && config["staging"]["enabled"]
    end

    def subscribe_to_staging
      options = {:do_not_track_subscription => true, :queue => "staging"}
      @staging_sid = nats.subscribe("staging", options) { |message| handle(message) }
    end

    def unsubscribe_from_staging
      nats.unsubscribe(@staging_sid) if @staging_sid
    end

    def subscribe_to_dea_specific_staging
      options = {:do_not_track_subscription => true}
      @dea_specified_staging_sid = nats.subscribe("staging.#{@dea_id}.start", options) { |message| handle(message) }
    end

    def unsubscribe_from_dea_specific_staging
      nats.unsubscribe(@dea_specified_staging_sid) if @dea_specified_staging_sid
    end

    def subscribe_to_staging_stop
      options = {:do_not_track_subscription => true}
      @staging_stop_sid = nats.subscribe("staging.stop", options) { |message| handle_stop(message) }
    end

    def unsubscribe_from_staging_stop
      nats.unsubscribe(@staging_stop_sid) if @staging_stop_sid
    end


    def respond_to_message(message, params)
      message.respond(
        "task_id" => params[:task_id],
        "task_streaming_log_url" => params[:streaming_log_url],
        "detected_buildpack" => params[:detected_buildpack],
        "error" => params[:error],
        "droplet_sha1" => params[:droplet_sha1]
      )
    end


    private

    def buildpacks_in_use
      staging_task_registry.flat_map do |task|
        task.admin_buildpacks
      end.uniq
    end
  end
end

*/
