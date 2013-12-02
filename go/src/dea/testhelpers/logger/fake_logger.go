package logger

import (
	steno "github.com/cloudfoundry/gosteno"
)

type FakeL struct {
	Logs map[string]string
}

func (f *FakeL) Level() steno.LogLevel {
	return steno.LOG_ALL
}

func (f *FakeL) Log(x steno.LogLevel, m string, d map[string]interface{}) {
	if f.Logs == nil {
		f.Logs = make(map[string]string)
	}
	f.Logs[x.Name] = m
}
