package dea

import (
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/nu7hatch/gouuid"
)

type Nats struct {
	messageBus cfmessagebus.MessageBus
}

type NatsHandler interface {
	HandleHealthManagerStart(payload []byte)
	HandleRouterStart(payload []byte)
	HandleDeaStatus(payload []byte)
	HandleDeaDirectedStart(payload []byte)
	HandleDeaStop(payload []byte)
	HandleDeaUpdate(payload []byte)
	HandleDeaFindDroplet(payload []byte)
	UUID() uuid.UUID
}

func NewNats(config NatsConfig) (*Nats, error) {
	messageBus, err := cfmessagebus.NewMessageBus("NATS")
	if err != nil {
		return nil, err
	}

	messageBus.Configure(config.Host, int(config.Port), config.User, config.Pass)

	return newNatsMBus(messageBus)
}

func newNatsMBus(mbus cfmessagebus.MessageBus) (*Nats, error) {
	return &Nats{messageBus: mbus}, nil
}

func (nats *Nats) Start(handler NatsHandler) error {
	mBus := nats.messageBus
	err := mBus.Connect()
	if err != nil {
		return err
	}

	if err = mBus.Subscribe("healthmanager.start", handler.HandleHealthManagerStart); err != nil {
		return err
	}

	if err = mBus.Subscribe("router.start", handler.HandleRouterStart); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.status", handler.HandleDeaStatus); err != nil {
		return err
	}

	uuid := handler.UUID()
	if err = mBus.Subscribe("dea."+uuid.String()+".start", handler.HandleDeaDirectedStart); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.stop", handler.HandleDeaStop); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.update", handler.HandleDeaUpdate); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.find.droplet", handler.HandleDeaFindDroplet); err != nil {
		return err
	}

	return nil
}
