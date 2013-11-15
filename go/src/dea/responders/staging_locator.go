package responders

import (
	"dea/config"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/yagnats"
	"time"
)

var stagingLocatorLogger = utils.Logger("StagingLocator", nil)

type StagingLocator struct {
	nats               yagnats.NATSClient
	id                 string
	resourceMgr        resmgr.ResourceManager
	advertiseIntervals time.Duration
	stacks             []string
	advertiseTicker    *time.Ticker
	staging_locate_sid int
}

func NewStagingLocator(nats yagnats.NATSClient, id string, resourceMgr resmgr.ResourceManager, config *config.Config) *StagingLocator {
	advertiseIntervals := default_advertise_interval
	if config.Intervals.Advertise != 0 {
		advertiseIntervals = config.Intervals.Advertise
	}

	return &StagingLocator{
		nats:               nats,
		id:                 id,
		resourceMgr:        resourceMgr,
		advertiseIntervals: advertiseIntervals,
		stacks:             config.Stacks,
	}
}

func (s *StagingLocator) Start() {
	sid, err := s.nats.Subscribe("staging.locate", s.handleIt)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}
	s.staging_locate_sid = sid

	s.advertiseTicker = utils.RepeatFixed(s.advertiseIntervals*time.Second, s.Advertise)
}

func (s *StagingLocator) Stop() {
	s.nats.Unsubscribe(s.staging_locate_sid)
	if s.advertiseTicker != nil {
		s.advertiseTicker.Stop()
	}
}

func (s *StagingLocator) handleIt(msg *yagnats.Message) {
	s.Advertise()
}

func (s *StagingLocator) Advertise() {
	am := protocol.NewAdvertiseMessage(s.id, s.stacks,
		s.resourceMgr.RemainingMemory(), s.resourceMgr.RemainingDisk(), nil)
	bytes, err := json.Marshal(am)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}
	s.nats.Publish("staging.advertise", bytes)
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
