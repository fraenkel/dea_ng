package starting

import (
	cfg "dea/config"
	cnr "dea/container"
	"errors"
	"github.com/cloudfoundry/gordon"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("StatCollector", func() {
	var container cnr.MockContainer
	var infoResponse warden.InfoResponse
	var collector StatCollector

	BeforeEach(func() {
		state := "state"
		cache := uint64(1)
		rss := uint64(2)
		bytes_used := uint64(42)
		usage := uint64(5000000)
		infoResponse = warden.InfoResponse{
			State: &state,
			MemoryStat: &warden.InfoResponse_MemoryStat{
				Cache: &cache,
				Rss:   &rss,
			},
			DiskStat: &warden.InfoResponse_DiskStat{
				BytesUsed: &bytes_used,
			},
			CpuStat: &warden.InfoResponse_CpuStat{
				Usage: &usage,
			},
		}

		container = cnr.MockContainer{}
		container.MInfoResponse = &infoResponse
	})

	AfterEach(func() {
		collector.Stop()
	})

	JustBeforeEach(func() {
		collector = NewStatCollector(&container)
	})

	It("has 0 used memory", func() {
		Expect(collector.GetStats().UsedMemory).To(Equal(cfg.Memory(0)))
	})

	It("has 0 used disk", func() {
		Expect(collector.GetStats().UsedDisk).To(Equal(cfg.Disk(0)))
	})

	It("has 0 computed pcu", func() {
		Expect(collector.GetStats().ComputedPCPU).To(Equal(float32(0)))
	})

	Describe("start", func() {
		Context("first time started", func() {
			It("retrieves stats", func() {
				collector.Start()
				Expect(collector.GetStats().UsedMemory).ToNot(Equal(cfg.Memory(0)))
			})

			It("runs #retrieve_stats every X seconds", func() {
				statCollector_INTERVAL = 30 * time.Millisecond
				collector.Start()
				time.Sleep(100 * time.Millisecond)

				Expect(len(collector.GetStats().cpu_samples)).To(Equal(2))
			})
		})

		Context("when already started", func() {
			It("returns false", func() {
				Expect(collector.Start()).To(BeTrue())
				Expect(collector.Start()).To(BeFalse())
			})
		})

	})

	Describe("stop", func() {
		Context("when already running", func() {
			It("stops the collector", func() {
				statCollector_INTERVAL = 30 * time.Millisecond
				collector.Start()
				collector.Stop()
				time.Sleep(50 * time.Millisecond)

				Expect(len(collector.GetStats().cpu_samples)).To(Equal(1))
			})
		})

		Context("when not running", func() {
			It("does nothing", func() {
				Expect(func() { collector.Stop() }).ToNot(Panic())
			})
		})
	})

	Describe("retrieve_stats", func() {
		Context("basic usage", func() {
			JustBeforeEach(func() {
				collector.Retrieve_stats(time.Now())
			})

			It("has 2Kb used memory", func() {
				Expect(collector.GetStats().UsedMemory).To(Equal(cfg.Memory(2 * cfg.Kibi)))
			})

			It("has 42 bytes used on disk", func() {
				Expect(collector.GetStats().UsedDisk).To(Equal(cfg.Disk(42)))
			})

			It("has 0 computed pcu", func() {
				Expect(collector.GetStats().ComputedPCPU).To(Equal(float32(0)))
			})

		})

		Context("when retrieving info fails", func() {
			BeforeEach(func() {
				container.MInfoError = errors.New("error")
			})

			It("does not propagate the error", func() {
				Expect(func() { collector.Retrieve_stats(time.Now()) }).ToNot(Panic())
			})

			It("keeps the same stats", func() {
				memory_before := collector.GetStats().UsedMemory
				disk_before := collector.GetStats().UsedDisk
				pcpu_before := collector.GetStats().ComputedPCPU

				collector.Retrieve_stats(time.Now())
				Expect(collector.GetStats().UsedMemory).To(Equal(memory_before))
				Expect(collector.GetStats().UsedDisk).To(Equal(disk_before))
				Expect(collector.GetStats().ComputedPCPU).To(Equal(pcpu_before))
			})
		})

		Context("and a second CPU sample comes in", func() {

			It("uses it to compute CPU usage", func() {
				now := time.Now()
				collector.Retrieve_stats(now)
				usage := uint64(10000000000)
				container.MInfoResponse.CpuStat.Usage = &usage
				collector.Retrieve_stats(now.Add(statCollector_INTERVAL))

				Expect(collector.GetStats().ComputedPCPU).To(Equal(float32((usage - 5000000) / uint64(statCollector_INTERVAL))))
			})
		})
	})
})
