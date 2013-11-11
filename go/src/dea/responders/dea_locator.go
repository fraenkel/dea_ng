package responders

import (
	"dea/config"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/go_cfmessagebus"
	"time"
)

var dealocatorLogger = utils.Logger("DeaLocator", nil)

type DeaLocator struct {
	mbus               cfmessagebus.MessageBus
	id                 string
	resourceMgr        *resmgr.ResourceManager
	advertiseIntervals time.Duration
	stacks             []string
	advertiseTicker    *time.Ticker
}

func NewDeaLocator(mbus cfmessagebus.MessageBus, id string, resourceMgr *resmgr.ResourceManager, config *config.Config) *DeaLocator {
	advertiseIntervals := default_advertise_interval
	if interval, exists := config.Intervals["advertise"]; exists {
		advertiseIntervals = interval
	}

	return &DeaLocator{
		mbus:               mbus,
		id:                 id,
		resourceMgr:        resourceMgr,
		advertiseIntervals: advertiseIntervals,
		stacks:             config.Stacks,
	}
}

func (d DeaLocator) Start() {
	if err := d.mbus.Subscribe("dea.locate", d.handleIt); err != nil {
		dealocatorLogger.Error(err.Error())
		return
	}

	d.advertiseTicker = utils.RepeatFixed(d.advertiseIntervals*time.Second, d.Advertise)
}
func (d DeaLocator) Stop() {
	//nats.unsubscribe(@dea_locate_sid) if @dea_locate_sid
	d.advertiseTicker.Stop()
}

func (d DeaLocator) handleIt(payload []byte) {
	d.Advertise()
}

func (d DeaLocator) Advertise() {
	am := protocol.NewAdvertiseMessage(d.id, d.stacks,
		d.resourceMgr.RemainingMemory(), d.resourceMgr.RemainingDisk(),
		d.resourceMgr.AppIdToCount())
	bytes, err := json.Marshal(am)
	if err != nil {
		dealocatorLogger.Error(err.Error())
		return
	}
	d.mbus.Publish("dea.advertise", bytes)
}
