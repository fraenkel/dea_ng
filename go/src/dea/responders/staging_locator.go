package responders

import (
	"dea"
	"dea/config"
	"dea/protocol"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/go_cfmessagebus"
	"time"
)

var stagingLocatorLogger = utils.Logger("StagingLocator", nil)

type StagingLocator struct {
	mbus               cfmessagebus.MessageBus
	id                 string
	resourceMgr        *dea.ResourceManager
	advertiseIntervals time.Duration
	stacks             []string
	advertiseTicker    *time.Ticker
}

func NewStagingLocator(mbus cfmessagebus.MessageBus, id string, resourceMgr *dea.ResourceManager, config *config.Config) *StagingLocator {
	advertiseIntervals := default_advertise_interval
	if interval, exists := config.Intervals["advertise"]; exists {
		advertiseIntervals = interval
	}

	return &StagingLocator{
		mbus:               mbus,
		id:                 id,
		resourceMgr:        resourceMgr,
		advertiseIntervals: advertiseIntervals,
		stacks:             config.Stacks,
	}
}

func (s StagingLocator) Start() {
	if err := s.mbus.Subscribe("staging.locate", s.handleIt); err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}
	s.advertiseTicker = utils.Repeat(s.Advertise, s.advertiseIntervals*time.Second)
}

func (s StagingLocator) Stop() {
	//nats.unsubscribe(@dea_locate_sid) if @dea_locate_sid
	s.advertiseTicker.Stop()
}

func (s StagingLocator) handleIt(payload []byte) {
	s.Advertise()
}

func (s StagingLocator) Advertise() {
	am := protocol.NewAdvertiseMessage(s.id, s.stacks,
		s.resourceMgr.RemainingMemory(), nil)
	bytes, err := json.Marshal(am)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}
	s.mbus.Publish("staging.advertise", bytes)
}

/*
module Dea::Responders
  class StagingLocator


    attr_reader :nats
    attr_reader :dea_id
    attr_reader :resource_manager
    attr_reader :config

    def initialize(nats, dea_id, resource_manager, config)
      @nats = nats
      @dea_id = dea_id
      @resource_manager = resource_manager
      @config = config
    end

    def start
      subscribe_to_staging_locate
      start_periodic_staging_advertise
    end

    def stop
      unsubscribe_from_staging_locate
      stop_periodic_staging_advertise
    end

    def advertise
      # Currently we are not tracking memory used by
      # staging task, therefore, available_memory
      # is not accurate since it only account for running apps.
      nats.publish("staging.advertise", {
        "id" => dea_id,
        "stacks" => config["stacks"],
        "available_memory" => resource_manager.remaining_memory
      })
    end

    private

    def subscribe_to_staging_locate
      options = {:do_not_track_subscription => true}
      @staging_locate_sid = nats.subscribe("staging.locate", options) { |_| advertise }
    end

    def unsubscribe_from_staging_locate
      nats.unsubscribe(@staging_locate_sid) if @staging_locate_sid
    end

    # Cloud controller uses staging.advertise to
    # keep track of all deas that it can use to run apps
    def start_periodic_staging_advertise
      advertise_interval = config["intervals"]["advertise"] || DEFAULT_ADVERTISE_INTERVAL
      @advertise_timer = EM.add_periodic_timer(advertise_interval) { advertise }
    end

    def stop_periodic_staging_advertise
      EM.cancel_timer(@advertise_timer) if @advertise_timer
    end
  end
end
*/
