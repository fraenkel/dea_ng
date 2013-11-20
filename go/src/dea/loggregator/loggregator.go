package loggregator

import (
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
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
		theEmitter.EmitLogMessage(theEmitter.NewLogMessage(appId, message, logmessage.LogMessage_ERR))
	}
}

func StagingEmit(appId, message string) {
	if theStagingEmitter != nil && appId != "" {
		theStagingEmitter.Emit(appId, message)
	}
}

func StagingEmitError(appId, message string) {
	if theStagingEmitter != nil && appId != "" {
		theStagingEmitter.EmitLogMessage(theStagingEmitter.NewLogMessage(appId, message, logmessage.LogMessage_ERR))
	}
}
