package emitter

import (
	logm "github.com/cloudfoundry/loggregatorlib/logmessage"
)

type MockEmitter struct {
	Messages      map[string][]string
	ErrorMessages map[string][]string
}

func (e *MockEmitter) Emit(appId, msg string) {
	if e.Messages == nil {
		e.Messages = make(map[string][]string)
	}
	msgs := e.Messages[appId]
	if msgs == nil {
		msgs = make([]string, 0, 5)
	}
	e.Messages[appId] = append(msgs, msg)
}

func (e *MockEmitter) EmitError(appId, msg string) {
	if e.ErrorMessages == nil {
		e.ErrorMessages = make(map[string][]string)
	}
	msgs := e.ErrorMessages[appId]
	if msgs == nil {
		msgs = make([]string, 0, 5)
	}
	e.ErrorMessages[appId] = append(msgs, msg)
}

func (e *MockEmitter) EmitLogMessage(logmsg *logm.LogMessage) {
	switch logmsg.GetMessageType() {
	case logm.LogMessage_OUT:
		e.Emit(logmsg.GetAppId(), string(logmsg.GetMessage()))

	case logm.LogMessage_ERR:
		e.EmitError(logmsg.GetAppId(), string(logmsg.GetMessage()))
	}

}
