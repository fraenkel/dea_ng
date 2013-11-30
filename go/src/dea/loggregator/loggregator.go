package loggregator

import (
	"github.com/cloudfoundry/loggregatorlib/emitter"
)

var theEmitter emitter.Emitter
var theStagingEmitter emitter.Emitter

func SetEmitter(e emitter.Emitter) {
	theEmitter = e
}

func SetStagingEmitter(e emitter.Emitter) {
	theStagingEmitter = e
}

func Emit(appId, message string) {
	if theEmitter != nil && appId != "" {
		theEmitter.Emit(appId, message)
	}
}

func EmitError(appId, message string) {
	if theEmitter != nil && appId != "" {
		theEmitter.EmitError(appId, message)
	}
}

func StagingEmit(appId, message string) {
	if theStagingEmitter != nil && appId != "" {
		theStagingEmitter.Emit(appId, message)
	}
}

func StagingEmitError(appId, message string) {
	if theStagingEmitter != nil && appId != "" {
		theStagingEmitter.EmitError(appId, message)
	}
}
