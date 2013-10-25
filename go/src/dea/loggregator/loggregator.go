package loggregator

import (
	"github.com/cloudfoundry/loggregatorlib/emitter"
)

var theEmitter emitter.Emitter

func SetEmitter(e emitter.Emitter) {
	theEmitter = e
}

func Emit(appId, message string) {
	if theEmitter != nil && appId != "" {
		theEmitter.Emit(appId, message)
	}
}
