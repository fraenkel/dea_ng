package starting

import (
	"dea"
	"dea/config"
	"dea/container"
	"dea/utils"
	"time"
)

var statCollector_INTERVAL = 10 * time.Second

var logger = utils.Logger("StatCollector", nil)

type statCollector struct {
	container container.Container
	timer     *time.Timer
	stats     dea.Stats
}

func NewStatCollector(container container.Container) dea.StatCollector {
	collector := &statCollector{container: container,
		stats: dea.Stats{CpuSamples: make([]dea.CPUStat, 0, 2)},
	}

	return collector
}

func (s *statCollector) Start() bool {
	if s.timer != nil {
		return false
	}

	s.timer = utils.Repeat(statCollector_INTERVAL, func() { s.Retrieve_stats(time.Now()) })
	s.Retrieve_stats(time.Now())

	return true

}

func (s *statCollector) Stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *statCollector) GetStats() dea.Stats {
	return s.stats
}

func (s *statCollector) Retrieve_stats(now time.Time) {
	info, err := s.container.Info()
	if err != nil {
		logger.Errorf("stat-collector.info-retrieval.failed handle:%s error:%s",
			s.container.Handle(), err.Error())
		return
	}

	stats := dea.Stats{
		UsedMemory: config.Memory(*info.MemoryStat.Rss) * config.Kibi,
		UsedDisk:   config.Disk(*info.DiskStat.BytesUsed),
		CpuSamples: make([]dea.CPUStat, 0, 2),
	}
	s.compute_cpu_usage(&stats, *info.CpuStat.Usage, now)
	s.stats = stats
}

func (s *statCollector) compute_cpu_usage(stats *dea.Stats, usage uint64, now time.Time) {
	if len(s.stats.CpuSamples) > 0 {
		stats.CpuSamples = append(stats.CpuSamples, s.stats.CpuSamples[len(s.stats.CpuSamples)-1])
	}
	stats.CpuSamples = append(stats.CpuSamples, dea.CPUStat{now, usage})

	if len(stats.CpuSamples) == 2 {
		used := stats.CpuSamples[1].Usage - stats.CpuSamples[0].Usage
		elapsed := stats.CpuSamples[1].Timestamp.Sub(stats.CpuSamples[0].Timestamp)
		if elapsed > 0 {
			stats.ComputedPCPU = float32(used / uint64(elapsed.Nanoseconds()))
		}
	}
}
