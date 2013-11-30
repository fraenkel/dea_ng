package starting

import (
	"dea/config"
	"dea/container"
	"dea/utils"
	"time"
)

var statCollector_INTERVAL = 10 * time.Second

type cpu_stat struct {
	timestamp time.Time
	usage     uint64
}

var logger = utils.Logger("StatCollector", nil)

type Stats struct {
	UsedMemory   config.Memory
	UsedDisk     config.Disk
	ComputedPCPU float32
	cpu_samples  []cpu_stat
}

type StatCollector interface {
	Start() bool
	Stop()
	GetStats() *Stats
	Retrieve_stats(now time.Time)
}

type statCollector struct {
	container container.Container
	timer     *time.Timer
	stats     *Stats
}

func NewStatCollector(container container.Container) StatCollector {
	collector := &statCollector{container: container,
		stats: &Stats{cpu_samples: make([]cpu_stat, 0, 2)},
	}

	return collector
}

func (s *statCollector) Start() bool {
	if s.timer != nil {
		return false
	}

	s.timer = utils.Repeat(statCollector_INTERVAL, func() { s.run_stat_collector() })

	s.run_stat_collector()

	return true

}

func (s *statCollector) Stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *statCollector) run_stat_collector() {
	s.Retrieve_stats(time.Now())
	if s.timer == nil {
		s.Start()
	}
}

func (s *statCollector) GetStats() *Stats {
	return s.stats
}

func (s *statCollector) Retrieve_stats(now time.Time) {
	info, err := s.container.Info()
	if err != nil {
		logger.Errorf("stat-collector.info-retrieval.failed handle:%s error:%s",
			s.container.Handle(), err.Error())
		return
	}

	stats := Stats{
		UsedMemory:  config.Memory(*info.MemoryStat.Rss) * config.Kibi,
		UsedDisk:    config.Disk(*info.DiskStat.BytesUsed),
		cpu_samples: make([]cpu_stat, 0, 2),
	}
	s.compute_cpu_usage(&stats, *info.CpuStat.Usage, now)
	s.stats = &stats
}

func (s *statCollector) compute_cpu_usage(stats *Stats, usage uint64, now time.Time) {
	if len(s.stats.cpu_samples) > 0 {
		stats.cpu_samples = append(stats.cpu_samples, s.stats.cpu_samples[len(s.stats.cpu_samples)-1])
	}
	stats.cpu_samples = append(stats.cpu_samples, cpu_stat{now, usage})

	if len(stats.cpu_samples) == 2 {
		used := stats.cpu_samples[1].usage - stats.cpu_samples[0].usage
		elapsed := stats.cpu_samples[1].timestamp.Sub(stats.cpu_samples[0].timestamp)
		if elapsed > 0 {
			stats.ComputedPCPU = float32(used / uint64(elapsed.Nanoseconds()))
		}
	}
}
