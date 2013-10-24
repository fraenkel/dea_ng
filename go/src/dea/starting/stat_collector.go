package starting

import (
	"dea/config"
	"dea/container"
	"dea/utils"
	"time"
)

const StatCollector_INTERVAL = 10

type cpu_stat struct {
	timestamp time.Time
	usage     uint64
}

type StatCollector struct {
	container    *container.Container
	ticker       *time.Ticker
	UsedMemory   config.Memory
	UsedDisk     config.Disk
	ComputedPCPU float32

	cpu_samples []cpu_stat
}

func NewStatCollector(container *container.Container) *StatCollector {
	collector := &StatCollector{container: container, cpu_samples: make([]cpu_stat, 0, 3)}

	return collector
}

func (s *StatCollector) start() bool {
	if s.ticker != nil {
		return false
	}

	s.ticker = utils.Repeat(func() { s.run_stat_collector() },
		StatCollector_INTERVAL)

	s.run_stat_collector()

	return true

}

func (s *StatCollector) stop() {
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

func (s *StatCollector) run_stat_collector() {
	s.retrieve_stats(time.Now())
	if s.ticker == nil {
		s.start()
	}
}

func (s *StatCollector) retrieve_stats(now time.Time) {
	info, err := s.container.Info()
	if err != nil {
		utils.Logger("StatCollector").Errorf("stat-collector.info-retrieval.failed handle:%s error:%s",
			s.container.GetHandle(), err.Error())
		return
	}

	s.UsedMemory = config.Memory(*info.MemoryStat.Rss) * config.Kibi
	s.UsedDisk = config.Disk(*info.DiskStat.BytesUsed)
	s.compute_cpu_usage(*info.CpuStat.Usage, now)
}

func (s *StatCollector) compute_cpu_usage(usage uint64, now time.Time) {
	s.cpu_samples = append(s.cpu_samples, cpu_stat{now, usage})
	if len(s.cpu_samples) > 2 {
		s.cpu_samples = s.cpu_samples[1:]
	}

	if len(s.cpu_samples) == 2 {
		used := s.cpu_samples[1].usage - s.cpu_samples[0].usage
		elapsed := s.cpu_samples[1].timestamp.Sub(s.cpu_samples[0].timestamp)
		if elapsed > 0 {
			s.ComputedPCPU = float32(used / uint64(elapsed.Nanoseconds()))
		}
	}
}
