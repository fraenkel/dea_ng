package resource_manager_test

import (
	"dea"
	cfg "dea/config"
	. "dea/resource_manager"
	"dea/staging"
	"dea/starting"
	"dea/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("ResourceManager", func() {
	var config cfg.Config
	var manager dea.ResourceManager
	var instanceRegistry dea.InstanceRegistry
	var stagingRegistry dea.StagingTaskRegistry
	var memory_mb uint64
	var memory_overcommit_factor float64
	var disk_mb uint64
	var disk_overcommit_factor float64

	nominal_memory_capacity := func() float64 {
		return float64(memory_mb) * memory_overcommit_factor
	}

	nominal_disk_capacity := func() float64 {
		return float64(disk_mb) * disk_overcommit_factor
	}

	createMemDiskInstance := func(mem, disk uint64, state dea.State) dea.Instance {
		attrs := testhelpers.Valid_instance_attributes(false)
		limits := attrs["limits"].(map[string]interface{})
		limits["mem"] = mem
		limits["disk"] = disk
		instance := starting.NewInstance(attrs, &config, nil, "127.0.0.1")
		instance.SetState(state)
		return instance
	}

	createInstance := func(amt uint64, state dea.State) dea.Instance {
		return createMemDiskInstance(amt, amt, state)
	}

	createStagingTask := func() dea.StagingTask {
		stgAttrs := testhelpers.Valid_staging_attributes()
		staging_message := staging.NewStagingMessage(stgAttrs)

		staging_task := staging.NewStagingTask(&config, staging_message,
			[]dea.StagingBuildpack{}, nil, nil)
		return staging_task
	}

	BeforeEach(func() {
		memory_mb = 600
		memory_overcommit_factor = 4
		disk_mb = 4000
		disk_overcommit_factor = 2
	})

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "resource_manager")
		config, _ = cfg.NewConfig(nil)
		config.BaseDir = tmpdir
		instanceRegistry = starting.NewInstanceRegistry(&config)
		stagingRegistry = staging.NewStagingTaskRegistry(&config, nil, staging.NewStagingTask)
	})

	AfterEach(func() {
		os.RemoveAll(config.BaseDir)
	})

	JustBeforeEach(func() {
		rConfig := cfg.ResourcesConfig{
			MemoryMB:               memory_mb,
			MemoryOvercommitFactor: memory_overcommit_factor,
			DiskMB:                 disk_mb,
			DiskOvercommitFactor:   disk_overcommit_factor,
		}
		manager = NewResourceManager(instanceRegistry, stagingRegistry, &rConfig)
	})

	Describe("remaining_memory", func() {
		Context("when no instances or staging tasks are registered", func() {
			It("returns the full memory capacity", func() {
				Expect(manager.RemainingMemory()).To(Equal(nominal_memory_capacity()))
			})
		})
		Context("when instances are registered", func() {

			BeforeEach(func() {
				instanceRegistry.Register(createInstance(1, dea.STATE_BORN))
				instanceRegistry.Register(createInstance(2, dea.STATE_STARTING))
				instanceRegistry.Register(createInstance(4, dea.STATE_RUNNING))
				instanceRegistry.Register(createInstance(8, dea.STATE_STOPPING))
				instanceRegistry.Register(createInstance(16, dea.STATE_STOPPED))
				instanceRegistry.Register(createInstance(32, dea.STATE_CRASHED))
				instanceRegistry.Register(createInstance(64, dea.STATE_DELETED))

				stgAttrs := testhelpers.Valid_staging_attributes()
				staging_message := staging.NewStagingMessage(stgAttrs)

				staging_task := staging.NewStagingTask(&config, staging_message,
					[]dea.StagingBuildpack{}, nil, nil)

				stagingRegistry.Register(staging_task)
			})

			It("returns the correct remaining memory", func() {
				Expect(manager.RemainingMemory()).To(BeNumerically("~", nominal_memory_capacity()-float64((1+2+4+8+config.Staging.MemoryLimitMB))))
			})
		})

	})

	Describe("remaining_disk", func() {
		Context("when no instances or staging tasks are registered", func() {
			It("returns the full disk capacity", func() {
				Expect(manager.RemainingDisk()).To(Equal(nominal_disk_capacity()))
			})
		})
		Context("when instances are registered", func() {

			BeforeEach(func() {
				instanceRegistry.Register(createInstance(1, dea.STATE_BORN))
				instanceRegistry.Register(createInstance(2, dea.STATE_STARTING))
				instanceRegistry.Register(createInstance(4, dea.STATE_RUNNING))
				instanceRegistry.Register(createInstance(8, dea.STATE_STOPPING))
				instanceRegistry.Register(createInstance(16, dea.STATE_STOPPED))
				instanceRegistry.Register(createInstance(32, dea.STATE_CRASHED))
				instanceRegistry.Register(createInstance(64, dea.STATE_DELETED))

				stagingRegistry.Register(createStagingTask())
			})

			It("returns the correct remaining memory", func() {
				Expect(manager.RemainingDisk()).To(BeNumerically("~", nominal_disk_capacity()-float64((1+2+4+8+32+config.Staging.DiskLimitMB))))
			})
		})

	})

	Describe("app_id_to_count", func() {
		createAppInstance := func(appId string, state dea.State) *starting.Instance {
			attrs := testhelpers.Valid_instance_attributes(false)
			attrs["application_id"] = appId
			instance := starting.NewInstance(attrs, &config, nil, "127.0.0.1")
			instance.SetState(state)
			return instance
		}

		BeforeEach(func() {
			instanceRegistry.Register(createAppInstance("a", dea.STATE_BORN))
			instanceRegistry.Register(createAppInstance("b", dea.STATE_STARTING))
			instanceRegistry.Register(createAppInstance("b", dea.STATE_STARTING))
			instanceRegistry.Register(createAppInstance("c", dea.STATE_RUNNING))
			instanceRegistry.Register(createAppInstance("c", dea.STATE_RUNNING))
			instanceRegistry.Register(createAppInstance("c", dea.STATE_RUNNING))
			instanceRegistry.Register(createAppInstance("d", dea.STATE_STOPPING))
			instanceRegistry.Register(createAppInstance("e", dea.STATE_STOPPED))
			instanceRegistry.Register(createAppInstance("f", dea.STATE_CRASHED))
			instanceRegistry.Register(createAppInstance("g", dea.STATE_DELETED))
		})

		It("should return all registered instances regardless of state", func() {
			expected := map[string]int{
				"a": 1,
				"b": 2,
				"c": 3,
				"d": 1,
				"e": 1,
				"f": 1,
				"g": 1,
			}
			Expect(manager.AppIdToCount()).To(Equal(expected))
		})
	})

	Describe("number_reservable", func() {
		BeforeEach(func() {
			memory_mb = 600
			memory_overcommit_factor = 1
			disk_mb = 4000
			disk_overcommit_factor = 1
		})

		Context("when there is not enough memory to reserve any", func() {
			It("is 0", func() {
				Expect(manager.NumberReservable(10000, 1)).To(BeNumerically("==", 0))
			})
		})

		Context("when there is not enough disk to reserve any", func() {
			It("is 0", func() {
				Expect(manager.NumberReservable(1, 10000)).To(BeNumerically("==", 0))
			})
		})

		Context("when there are enough resources for a single reservation", func() {
			It("is 1", func() {
				Expect(manager.NumberReservable(500, 3000)).To(BeNumerically("==", 1))
			})
		})

		Context("when there are enough resources for many reservations", func() {
			It("is correct", func() {
				Expect(manager.NumberReservable(200, 1500)).To(BeNumerically("==", 2))
				Expect(manager.NumberReservable(200, 1000)).To(BeNumerically("==", 3))
			})
		})

		Context("when 0 resources are requested", func() {
			It("returns 0", func() {
				Expect(manager.NumberReservable(0, 0)).To(BeNumerically("==", 0))
			})
		})
	})

	Describe("available_memory_ratio", func() {
		BeforeEach(func() {
			instanceRegistry.Register(createInstance(512, dea.STATE_RUNNING))
			stagingRegistry.Register(createStagingTask())

		})

		It("is the ratio of available memory to total memory", func() {
			Expect(manager.AvailableMemoryRatio()).To(BeNumerically("~", 1-float64(512+config.Staging.MemoryLimitMB)/nominal_memory_capacity()))
		})
	})

	Describe("available_disk_ratio", func() {
		BeforeEach(func() {
			instanceRegistry.Register(createInstance(512, dea.STATE_RUNNING))
			stagingRegistry.Register(createStagingTask())
		})

		It("is the ratio of available disk to total disk", func() {
			Expect(manager.AvailableDiskRatio()).To(BeNumerically("~", 1-float64(512+config.Staging.DiskLimitMB)/nominal_disk_capacity()))
		})
	})

	Describe("check if there are available resources", func() {
		Describe("could_reserve?", func() {
			var remaining_memory float64
			var remaining_disk float64

			BeforeEach(func() {
				instanceRegistry.Register(createMemDiskInstance(512, 1024, dea.STATE_RUNNING))
				stagingRegistry.Register(createStagingTask())

				remaining_memory = nominal_memory_capacity() - float64(512+config.Staging.MemoryLimitMB)
				remaining_disk = nominal_disk_capacity() - float64(1024+config.Staging.DiskLimitMB)
			})

			Context("when the given amounts of memory and disk are available (including extra 'headroom' memory)", func() {
				It("can reserve", func() {
					Expect(manager.CanReserve(remaining_memory-1, remaining_disk-1)).To(BeTrue())
				})
			})

			Context("when too much memory is being used", func() {
				It("can't reserve", func() {
					Expect(manager.CanReserve(remaining_memory+1, 1)).To(BeFalse())
				})
			})

			Context("when too much disk is being used", func() {
				It("can't reserve", func() {
					Expect(manager.CanReserve(1, remaining_disk+1)).To(BeFalse())
				})
			})
		})
	})
})
