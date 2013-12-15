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

var dealocatorLogger = utils.Logger("DeaLocator", nil)

type DeaLocator struct {
	nats                yagnats.NATSClient
	subscriptionId      int
	id                  string
	resourceMgr         dea.ResourceManager
	advertiseIntervals  time.Duration
	stacks              []string
	placementProperties config.PlacementConfig
	advertiseTicker     *time.Ticker
}

func NewDeaLocator(nats yagnats.NATSClient, id string, resourceMgr dea.ResourceManager, config *config.Config) *DeaLocator {
	advertiseIntervals := default_advertise_interval
	if config.Intervals.Advertise != 0 {
		advertiseIntervals = config.Intervals.Advertise
	}

	return &DeaLocator{
		nats:                nats,
		id:                  id,
		resourceMgr:         resourceMgr,
		advertiseIntervals:  advertiseIntervals,
		stacks:              config.Stacks,
		placementProperties: config.PlacementProperties,
	}
}

func (d *DeaLocator) Start() {
	sid, err := d.nats.Subscribe("dea.locate", d.handleIt)
	if err != nil {
		dealocatorLogger.Error(err.Error())
		return
	}
	d.subscriptionId = sid
	d.advertiseTicker = utils.RepeatFixed(d.advertiseIntervals, d.Advertise)
}

func (d *DeaLocator) Stop() {
	if d.advertiseTicker != nil {
		d.nats.Unsubscribe(d.subscriptionId)
		d.advertiseTicker.Stop()
	}
}

func (d *DeaLocator) handleIt(msg *yagnats.Message) {
	d.Advertise()
}

func (d *DeaLocator) Advertise() {
	am := protocol.NewAdvertiseMessage(d.id, d.stacks,
		d.resourceMgr.RemainingMemory(), d.resourceMgr.RemainingDisk(),
		d.resourceMgr.AppIdToCount(),
		d.placementProperties)

	bytes, err := json.Marshal(am)
	if err != nil {
		dealocatorLogger.Error(err.Error())
		return
	}
	d.nats.Publish("dea.advertise", bytes)
}
