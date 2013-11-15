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
		s.resourceMgr.RemainingMemory(), s.resourceMgr.RemainingDisk(), nil)
	bytes, err := json.Marshal(am)
	if err != nil {
		stagingLocatorLogger.Error(err.Error())
		return
	}

	s.nats.Publish("staging.advertise", bytes)
}
