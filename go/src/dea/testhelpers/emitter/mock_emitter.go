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

func (e *MockEmitter) EmitLogMessage(logmsg *logm.LogMessage) {
	var msgMap map[string][]string

	switch logmsg.GetMessageType() {
	case logm.LogMessage_OUT:
		if e.Messages == nil {
			e.Messages = make(map[string][]string)
		}
		msgMap = e.Messages

	case logm.LogMessage_ERR:
		if e.ErrorMessages == nil {
			e.ErrorMessages = make(map[string][]string)
		}
		msgMap = e.ErrorMessages
	}

	appId := logmsg.GetAppId()
	msgs := msgMap[appId]
	if msgs == nil {
		msgs = make([]string, 0, 5)
	}

	msgMap[appId] = append(msgs, string(logmsg.GetMessage()))
}

func (e *MockEmitter) NewLogMessage(appId, message string, mt logm.LogMessage_MessageType) *logm.LogMessage {
	return &logm.LogMessage{AppId: &appId, Message: []byte(message), MessageType: &mt}
}
