package starting

import (
	"dea"
	"time"
)

type FakeStatCollector struct {
	Stats dea.Stats
}

func (fsc *FakeStatCollector) Start() bool {
	return true
}
func (fsc *FakeStatCollector) Stop() {
}
func (fsc *FakeStatCollector) GetStats() dea.Stats {
	return fsc.Stats
}
func (fsc *FakeStatCollector) Retrieve_stats(now time.Time) {
}
