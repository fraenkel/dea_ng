package staging

import (
	cfg "dea/config"
	"dea/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/url"
	"os"
)

var _ = Describe("StagingTaskRegistry", func() {
	var config cfg.Config

	var task1 StagingTask
	var task2 StagingTask
	var task1Attrs map[string]interface{}
	var task2Attrs map[string]interface{}

	var stagingRegistry *StagingTaskRegistry

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "staging_task_registry")
		config = cfg.Config{BaseDir: tmpdir,
			Staging: cfg.StagingConfig{MemoryLimitMB: 1024, DiskLimitMB: 2048},
		}
		task1Attrs = testhelpers.Valid_staging_attributes()
		task2Attrs = testhelpers.Valid_staging_attributes()
	})

	AfterEach(func() {
		os.RemoveAll(config.BaseDir)
	})

	JustBeforeEach(func() {
		stagingRegistry = NewStagingTaskRegistry(NewStagingTask)
		task1 = stagingRegistry.NewStagingTask(&config, NewStagingMessage(task1Attrs), nil, nil)
		task2 = stagingRegistry.NewStagingTask(&config, NewStagingMessage(task2Attrs), nil, nil)
	})

	Describe("Register", func() {
		It("adds to the registry", func() {
			stagingRegistry.Register(task1)
			Expect(len(stagingRegistry.Tasks())).To(Equal(1))
		})
	})

	Describe("unregister", func() {
		Context("when task was previously registered", func() {

			It("removes task from the registry", func() {
				stagingRegistry.Register(task1)

				stagingRegistry.Unregister(task1)
				Expect(len(stagingRegistry.Tasks())).To(Equal(0))
			})
		})

		Context("when task was not previously registered", func() {
			It("does nothing", func() {
				stagingRegistry.Unregister(task1)
				Expect(len(stagingRegistry.Tasks())).To(Equal(0))
			})

		})
	})

	Describe("tasks", func() {
		It("returns a specified task", func() {
			stagingRegistry.Register(task1)

			Expect(stagingRegistry.Task(task1.Id())).To(Equal(task1))
		})

		It("returns all previously registered tasks", func() {
			stagingRegistry.Register(task1)
			stagingRegistry.Register(task2)

			tasks := stagingRegistry.Tasks()
			Expect(tasks).To(Equal([]StagingTask{task1, task2}))
		})
	})

	Describe("ReservedDisk", func() {
		Context("when no tasks registered", func() {
			It("returns 0", func() {
				Expect(stagingRegistry.ReservedDisk()).To(Equal(cfg.Disk(0)))
			})
		})
		Context("when multiple tasks registered", func() {
			It("returns a multiple based on config", func() {
				stagingRegistry.Register(task1)
				stagingRegistry.Register(task2)

				Expect(stagingRegistry.ReservedDisk()).To(Equal(cfg.Disk(2*config.Staging.DiskLimitMB) * cfg.MB))
			})
		})
	})

	Describe("ReservedMemory", func() {
		Context("when no tasks registered", func() {
			It("returns 0", func() {
				Expect(stagingRegistry.ReservedMemory()).To(Equal(cfg.Memory(0)))
			})

		})
		Context("when multiple tasks registered", func() {
			It("returns a multiple based on config", func() {
				stagingRegistry.Register(task1)
				stagingRegistry.Register(task2)

				Expect(stagingRegistry.ReservedMemory()).To(Equal(cfg.Memory(2*config.Staging.MemoryLimitMB) * cfg.Mebi))
			})
		})
	})

	Describe("BuildpacksInUse", func() {
		Context("when no tasks registered", func() {
			It("returns nothing", func() {
				Expect(len(stagingRegistry.BuildpacksInUse())).To(Equal(0))
			})
		})
		Context("when multiple tasks registered", func() {
			var buildpacks []StagingBuildpack
			BeforeEach(func() {
				u, _ := url.Parse("http://www.example.com")
				buildpacks = []StagingBuildpack{
					StagingBuildpack{Url: u, Key: "bp1"},
					StagingBuildpack{Url: u, Key: "bp2"},
					StagingBuildpack{Url: u, Key: "bp3"},
				}
				task1Attrs["admin_buildpacks"] = []map[string]interface{}{
					{"url": u.String(), "key": buildpacks[0].Key},
					{"url": u.String(), "key": buildpacks[1].Key}}
				task2Attrs["admin_buildpacks"] = []map[string]interface{}{
					{"url": u.String(), "key": buildpacks[1].Key},
					{"url": u.String(), "key": buildpacks[2].Key}}
			})

			It("returns the set of buildpacks", func() {
				stagingRegistry.Register(task1)
				stagingRegistry.Register(task2)

				Expect(stagingRegistry.BuildpacksInUse()).To(Equal(buildpacks))
			})
		})
	})

})
