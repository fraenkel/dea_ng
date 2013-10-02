package dea

import (
	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	"github.com/nu7hatch/gouuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateNats(t *testing.T) {
	_, err := NewNats(Config{}.NatsConfig)
	if err != nil {
		t.Fatal(err)
	}
}

type MockHandler struct {
	uuid           uuid.UUID
	handledMessage string
	handledPayload []byte
}

var lastMock MockHandler

func (mockHandler MockHandler) HandleHealthManagerStart(payload []byte) {
	mockHandler.handled("healthmanager.start", payload)
}
func (mockHandler MockHandler) HandleRouterStart(payload []byte) {
}
func (mockHandler MockHandler) HandleDeaStatus(payload []byte) {
}
func (mockHandler MockHandler) HandleDeaDirectedStart(payload []byte) {
}
func (mockHandler MockHandler) HandleDeaStop(payload []byte) {
}
func (mockHandler MockHandler) HandleDeaUpdate(payload []byte) {
}
func (mockHandler MockHandler) HandleDeaFindDroplet(payload []byte) {
}
func (mockHandler MockHandler) UUID() uuid.UUID {
	return mockHandler.uuid
}
func expect(t *testing.T, message string, payload []byte) {
	assert.Equal(t, message, lastMock.handledMessage)
	assert.Equal(t, payload, lastMock.handledPayload)
}

func (mockHandler MockHandler) handled(message string, payload []byte) {
	lastMock = mockHandler
	lastMock.handledMessage = message
	lastMock.handledPayload = payload
}

func TestStartNats(t *testing.T) {
	mbus := mock_cfmessagebus.NewMockMessageBus()
	nats, err := newNatsMBus(mbus)
	if err != nil {
		t.Fatal(err)
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	mockHandler := MockHandler{uuid: *uuid}
	assert.Nil(t, nats.Start(&mockHandler))

	payload := []byte{'a', 'b'}
	mbus.PublishSync("healthmanager.start", payload)
	expect(t, "healthmanager.start", payload)
}
