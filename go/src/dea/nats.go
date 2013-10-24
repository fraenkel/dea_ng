package dea

import (
	"dea/config"
	"github.com/cloudfoundry/go_cfmessagebus"
)

type Nats struct {
	MessageBus cfmessagebus.MessageBus
}

type NatsHandler interface {
	HandleHealthManagerStart(payload []byte)
	HandleRouterStart(payload []byte)
	HandleDeaStatus(payload []byte, replyTo cfmessagebus.ReplyTo)
	HandleDeaDirectedStart(payload []byte)
	HandleDeaStop(payload []byte)
	HandleDeaUpdate(payload []byte)
	HandleDeaFindDroplet(payload []byte, replyTo cfmessagebus.ReplyTo)
	UUID() string
}

func NewNats(config config.NatsConfig) (*Nats, error) {
	messageBus, err := cfmessagebus.NewMessageBus("NATS")
	if err != nil {
		return nil, err
	}

	messageBus.Configure(config.Host, int(config.Port), config.User, config.Pass)

	return newNatsMBus(messageBus)
}

func newNatsMBus(mbus cfmessagebus.MessageBus) (*Nats, error) {
	return &Nats{MessageBus: mbus}, nil
}

func (nats Nats) Start(handler NatsHandler) error {
	mBus := nats.MessageBus
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

	if err = mBus.ReplyToChannel("dea.status", handler.HandleDeaStatus); err != nil {
		return err
	}

	uuid := handler.UUID()
	if err = mBus.Subscribe("dea."+uuid+".start", handler.HandleDeaDirectedStart); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.stop", handler.HandleDeaStop); err != nil {
		return err
	}

	if err = mBus.Subscribe("dea.update", handler.HandleDeaUpdate); err != nil {
		return err
	}

	if err = mBus.ReplyToChannel("dea.find.droplet", handler.HandleDeaFindDroplet); err != nil {
		return err
	}

	return nil
}

func (nats Nats) Stop() {
	nats.MessageBus.Disconnect()
}
