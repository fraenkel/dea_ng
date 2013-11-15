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

var dealocatorLogger = utils.Logger("DeaLocator", nil)

type DeaLocator struct {
	nats               yagnats.NATSClient
	subscriptionId     int
	id                 string
	resourceMgr        resmgr.ResourceManager
	advertiseIntervals time.Duration
	stacks             []string
	advertiseTicker    *time.Ticker
}

func NewDeaLocator(nats yagnats.NATSClient, id string, resourceMgr resmgr.ResourceManager, config *config.Config) *DeaLocator {
	advertiseIntervals := default_advertise_interval
	if config.Intervals.Advertise != 0 {
		advertiseIntervals = config.Intervals.Advertise
	}

	return &DeaLocator{
		nats:               nats,
		id:                 id,
		resourceMgr:        resourceMgr,
		advertiseIntervals: advertiseIntervals,
		stacks:             config.Stacks,
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
		d.resourceMgr.AppIdToCount())
	bytes, err := json.Marshal(am)
	if err != nil {
		dealocatorLogger.Error(err.Error())
		return
	}
	d.nats.Publish("dea.advertise", bytes)
}
