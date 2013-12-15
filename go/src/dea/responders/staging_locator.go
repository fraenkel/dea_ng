package responders

import (
	"dea"
	"dea/config"
	"dea/protocol"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/yagnats"
	"time"
)

var stagingLocatorLogger = utils.Logger("StagingLocator", nil)

type StagingLocator struct {
	nats                yagnats.NATSClient
	id                  string
	resourceMgr         dea.ResourceManager
	advertiseIntervals  time.Duration
	stacks              []string
	placementProperties config.PlacementConfig
	advertiseTicker     *time.Ticker
	staging_locate_sid  int
}

func NewStagingLocator(nats yagnats.NATSClient, id string, resourceMgr dea.ResourceManager, config *config.Config) *StagingLocator {
	advertiseIntervals := default_advertise_interval
	if config.Intervals.Advertise != 0 {
		advertiseIntervals = config.Intervals.Advertise
	}

	return &StagingLocator{
		nats:                nats,
		id:                  id,
		resourceMgr:         resourceMgr,
		advertiseIntervals:  advertiseIntervals,
		stacks:              config.Stacks,
		placementProperties: config.PlacementProperties,
	}
}

func (s *StagingLocator) Start() {
	sid, err := s.nats.Subscribe("staging.locate", s.handleIt)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}
	s.staging_locate_sid = sid

	s.advertiseTicker = utils.RepeatFixed(s.advertiseIntervals, s.Advertise)
}

func (s *StagingLocator) Stop() {
	if s.advertiseTicker != nil {
		s.nats.Unsubscribe(s.staging_locate_sid)
		s.advertiseTicker.Stop()
	}
}

func (s *StagingLocator) handleIt(msg *yagnats.Message) {
	s.Advertise()
}

func (s *StagingLocator) Advertise() {
	am := protocol.NewAdvertiseMessage(s.id, s.stacks,
		s.resourceMgr.RemainingMemory(), s.resourceMgr.RemainingDisk(),
		s.resourceMgr.AppIdToCount(),
		s.placementProperties)

	bytes, err := json.Marshal(am)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}

	s.nats.Publish("staging.advertise", bytes)
}
