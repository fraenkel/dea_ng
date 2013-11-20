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

type StatCollector struct {
	container    container.Container
	timer        *time.Timer
	UsedMemory   config.Memory
	UsedDisk     config.Disk
	ComputedPCPU float32

	cpu_samples []cpu_stat
}

func NewStatCollector(container container.Container) *StatCollector {
	collector := &StatCollector{container: container, cpu_samples: make([]cpu_stat, 0, 3)}

	return collector
}

func (s *StatCollector) start() bool {
	if s.timer != nil {
		return false
	}

	s.timer = utils.Repeat(statCollector_INTERVAL, func() { s.run_stat_collector() })

	s.run_stat_collector()

	return true

}

func (s *StatCollector) stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *StatCollector) run_stat_collector() {
	s.retrieve_stats(time.Now())
	if s.timer == nil {
		s.start()
	}
}

func (s *StatCollector) retrieve_stats(now time.Time) {
	info, err := s.container.Info()
	if err != nil {
		logger.Errorf("stat-collector.info-retrieval.failed handle:%s error:%s",
			s.container.Handle(), err.Error())
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
