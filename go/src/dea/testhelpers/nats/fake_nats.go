package nats

import (
	"dea"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
)

type FakeNats struct {
	NatsClient   *fakeyagnats.FakeYagnats
	Unsubscribed bool
}

func NewFakeNats() *FakeNats {
	return &FakeNats{
		NatsClient: fakeyagnats.New(),
	}
}

func (fn *FakeNats) Client() yagnats.NATSClient {
	return fn.NatsClient
}

func (fn *FakeNats) Start(handler dea.NatsHandler) error {
	return nil
}

func (fn *FakeNats) Request(subject string, message []byte, callback yagnats.Callback) (int, error) {
	return 0, nil
}

func (fn *FakeNats) Unsubscribe() {
	fn.Unsubscribed = true
}
