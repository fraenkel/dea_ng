package lifecycle

import (
	"dea"
	"dea/protocol"
	"encoding/json"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	"time"
)

const EXIT_REASON_EVACUATION = "DEA_EVACUATION"

type EvacuationHandler interface {
	Evacuate(goodbyeMsg *protocol.GoodbyeMessage) bool
}

type evacuationHandler struct {
	evacuationProcessed bool
	endTime             time.Time
	evacuationBailOut   time.Duration
	responders          []dea.Responder
	instanceRegistry    dea.InstanceRegistry
	natsClient          yagnats.NATSClient
	logger              *steno.Logger
}

func NewEvacuationHandler(evacBailOut time.Duration, responders []dea.Responder, iRegistry dea.InstanceRegistry, natsClient yagnats.NATSClient, logger *steno.Logger) EvacuationHandler {
	return &evacuationHandler{
		evacuationBailOut: evacBailOut,
		responders:        responders,
		instanceRegistry:  iRegistry,
		natsClient:        natsClient,
		logger:            logger,
	}
}

func (e *evacuationHandler) Evacuate(goodbyeMsg *protocol.GoodbyeMessage) bool {
	if !e.isEvacuating() {
		e.endTime = time.Now().Add(e.evacuationBailOut)
	}

	e.logger.Infof("Evacuating (first time %s; can shutdown: %s", e.isEvacuating(), e.canDeaShutdown())

	if !e.isEvacuating() {
		stopResponders(e.responders)
		sendGoodbyeMessage(goodbyeMsg, e.natsClient, e.logger)
	}

	e.sendDropletExitedMessages()
	e.transitionInstancesToEvacuating()

	e.evacuationProcessed = true

	return e.canDeaShutdown()
}

func (e *evacuationHandler) isEvacuating() bool {
	return e.evacuationProcessed
}

func (e *evacuationHandler) sendDropletExitedMessages() {
	if e.instanceRegistry != nil {
		for _, i := range e.instanceRegistry.Instances() {
			state := i.State()
			switch state {
			case dea.STATE_BORN, dea.STATE_STARTING, dea.STATE_RESUMING, dea.STATE_RUNNING:
				e.sendExitMessage(i, EXIT_REASON_EVACUATION)
			}
		}
	}
}

func (e *evacuationHandler) sendExitMessage(i dea.Instance, reason string) {
	exitm := protocol.NewExitMessage(i, reason)
	bytes, err := json.Marshal(exitm)
	if err != nil {
		e.logger.Error(err.Error())
		return
	}
	e.natsClient.Publish("droplet.exited", bytes)
}

func (e *evacuationHandler) transitionInstancesToEvacuating() {
	if e.instanceRegistry != nil {
		for _, i := range e.instanceRegistry.Instances() {
			state := i.State()
			switch state {
			case dea.STATE_BORN, dea.STATE_STARTING, dea.STATE_RESUMING, dea.STATE_RUNNING:
				if e.isEvacuating() {
					e.logger.Errorf("Found an unexpected %s instance while evacuating", i.State())
				}
				i.SetState(dea.STATE_EVACUATING)
			}
		}
	}
}

func (e *evacuationHandler) canDeaShutdown() bool {
	if !e.endTime.Before(time.Now()) {
		return true
	}

	for _, i := range e.instanceRegistry.Instances() {
		switch i.State() {
		case dea.STATE_STOPPING, dea.STATE_STOPPED, dea.STATE_CRASHED, dea.STATE_DELETED:
		default:
			return false
		}
	}

	return true
}
