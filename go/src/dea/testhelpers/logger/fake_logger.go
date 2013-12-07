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

func RegisterFakeSink() *FakeSink {
	f := FakeSink{
		Records: make(map[steno.LogLevel][]*steno.Record),
	}

	c := steno.Config{
		Sinks: []steno.Sink{&f},
		Level: steno.LOG_ALL,
	}
	steno.Init(&c)
	return &f
}

type FakeSink struct {
	Records map[steno.LogLevel][]*steno.Record
}

func (fs *FakeSink) AddRecord(r *steno.Record) {
	recs := fs.Records[r.Level]
	if recs == nil {
		recs = make([]*steno.Record, 0, 1)
	}
	fs.Records[r.Level] = append(recs, r)
}

func (fs *FakeSink) Flush() {
}

func (fs *FakeSink) SetCodec(codec steno.Codec) {
}
func (fs *FakeSink) GetCodec() steno.Codec {
	return nil
}
