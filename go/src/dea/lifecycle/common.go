package lifecycle

import (
	"dea"
	"dea/protocol"
	"encoding/json"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
)

func stopResponders(responders []dea.Responder) {
	for _, r := range responders {
		r.Stop()
	}
}

func sendGoodbyeMessage(goodbyeMsg *protocol.GoodbyeMessage, nats yagnats.NATSClient, logger *steno.Logger) {
	bytes, err := json.Marshal(goodbyeMsg)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	nats.Publish("dea.shutdown", bytes)
}
