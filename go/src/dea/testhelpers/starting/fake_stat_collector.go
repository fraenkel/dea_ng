package starting

import (
	"dea/starting"
	"time"
)

type FakeStatCollector struct {
	Stats starting.Stats
}

func (fsc *FakeStatCollector) Start() bool {
	return true
}
func (fsc *FakeStatCollector) Stop() {
}
func (fsc *FakeStatCollector) GetStats() starting.Stats {
	return fsc.Stats
}
func (fsc *FakeStatCollector) Retrieve_stats(now time.Time) {
}
