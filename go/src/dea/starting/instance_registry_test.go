package starting

import (
	"dea"
	cfg "dea/config"
	"dea/droplet"
	"dea/loggregator"
	thelpers "dea/testhelpers"
	temitter "dea/testhelpers/emitter"
	"dea/utils"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"
)

var _ = Describe("InstanceRegistry", func() {
	var config cfg.Config
	var dropletRegistry dea.DropletRegistry
	var instance_registry *instanceRegistry

	var attributes map[string]interface{}
	var attributes1 map[string]interface{}
	var instance *Instance
	var instance1 *Instance

	BeforeEach(func() {
		config = cfg.Config{}
		dropletRegistry = droplet.NewDropletRegistry(config.BaseDir)

		attributes = thelpers.Valid_instance_attributes(false)
		attributes1 = thelpers.Valid_instance_attributes(false)
	})

	JustBeforeEach(func() {
		instance = NewInstance(attributes, &config, dropletRegistry, "127.0.0.1")
		instance1 = NewInstance(attributes1, &config, dropletRegistry, "127.0.0.1")
		instance_registry = NewInstanceRegistry(&config).(*instanceRegistry)
	})

	Describe("change_instance_id", func() {
		var old_instance_id string

		JustBeforeEach(func() {
			old_instance_id = instance.Id()
			instance_registry.Register(instance)
			instance_registry.ChangeInstanceId(instance)
		})

		It("should change the instance_id on the instance", func() {
			Expect(instance.Id()).ToNot(Equal(old_instance_id))
		})

		It("should return the instance when querying against the new instance_id", func() {
			Expect(instance_registry.LookupInstance(old_instance_id)).To(BeNil())
			Expect(instance_registry.LookupInstance(instance.Id())).To(Equal(instance))
		})

		Context("when looking up by application_id, the instances have the correct changed id", func() {
			It("should rearrange the by_application cache", func() {
				instances := instance_registry.InstancesForApplication(instance.ApplicationId())
				Expect(instances).To(HaveLen(1))
				Expect(instances[instance.Id()]).To(Equal(instance))
			})
		})
	})

	Describe("register", func() {
		It("should allow one to lookup the instance by id", func() {
			instance_registry.Register(instance)
			Expect(instance_registry.LookupInstance(instance.Id())).To(Equal(instance))
		})

		It("should allow one to lookup the instance by application id", func() {
			instance_registry.Register(instance)
			instances := instance_registry.InstancesForApplication(instance.ApplicationId())
			Expect(instances).To(HaveLen(1))
			Expect(instances[instance.Id()]).To(Equal(instance))
		})

		It("should log to the loggregator", func() {
			fakeEmitter := temitter.FakeEmitter{}
			loggregator.SetEmitter(&fakeEmitter)

			instance_registry.Register(instance)
			Expect(fakeEmitter.Messages).To(HaveLen(1))
			Expect(fakeEmitter.Messages[instance.ApplicationId()][0]).To(Equal("Registering instance"))
		})
	})

	Describe("unregister", func() {
		JustBeforeEach(func() {
			instance_registry.Register(instance)
		})

		It("should ensure the instance cannot be looked up by id", func() {
			instance_registry.Unregister(instance)
			Expect(instance_registry.LookupInstance(instance.Id())).To(BeNil())
		})

		It("should ensure the instance cannot be looked up by application id", func() {
			instance_registry.Unregister(instance)
			instances := instance_registry.InstancesForApplication(instance.ApplicationId())
			Expect(instances).To(BeNil())
		})

		It("should log to the loggregator", func() {
			fakeEmitter := temitter.FakeEmitter{}
			loggregator.SetEmitter(&fakeEmitter)

			instance_registry.Unregister(instance)
			Expect(fakeEmitter.Messages).To(HaveLen(1))
			Expect(fakeEmitter.Messages[instance.ApplicationId()][0]).To(Equal("Removing instance"))
		})
	})

	Describe("instances_for_application", func() {
		JustBeforeEach(func() {
			instance_registry.Register(instance)
			instance_registry.Register(instance1)
		})

		It("should return all registered instances for the supplied application id", func() {
			instances := instance_registry.InstancesForApplication(instance.ApplicationId())
			Expect(instances).To(HaveLen(2))
			Expect(instances).To(Equal(map[string]dea.Instance{
				instance.Id():  instance,
				instance1.Id(): instance1,
			}))
		})
	})

	Describe("app_id_to_count", func() {
		Context("when there are no instances", func() {
			It("is an empty map", func() {
				Expect(instance_registry.AppIdToCount()).To(HaveLen(0))
			})
		})

		Context("when there are instances", func() {
			JustBeforeEach(func() {
				instance_registry.Register(instance)
				instance_registry.Register(instance1)
			})

			It("is a map of the number of instances per app id", func() {
				Expect(instance_registry.AppIdToCount()).To(Equal(map[string]int{
					instance.ApplicationId(): 2}))
			})
		})
	})

	Describe("instances", func() {
		JustBeforeEach(func() {
			instance_registry.Register(instance)
			instance_registry.Register(instance1)
		})

		It("should return all registered instances", func() {
			Expect(instance_registry.Instances()).Should(HaveLen(2))
			Expect(instance_registry.Instances()).Should(ContainElement(instance))
			Expect(instance_registry.Instances()).Should(ContainElement(instance1))
		})
	})

	is_reaped := func(instance dea.Instance) bool {
		crash_path := path.Join(config.CrashesPath, instance.Id())
		return !utils.File_Exists(crash_path)
	}

	register_crashed_instance := func(registry dea.InstanceRegistry, options map[string]interface{}) *Instance {
		attrs := thelpers.Valid_instance_attributes(false)
		if options != nil {
			for k, v := range options {
				attrs[k] = v
			}
		}

		instance = NewInstance(attrs, &config, dropletRegistry, "127.0.0.1")

		instance.SetState(dea.STATE_CRASHED)

		crash_path := path.Join(config.CrashesPath, instance.Id())

		os.MkdirAll(crash_path, 0755)

		if registry != nil {
			registry.Register(instance)
		}

		return instance
	}

	Describe("instances_filtered_by_message", func() {
		merge := func(m map[string]interface{}, adds map[string]interface{}) {
			for k, v := range adds {
				m[k] = v
			}
		}

		BeforeEach(func() {
			merge(attributes, map[string]interface{}{
				"application_id":      "1",
				"application_version": "abc",
				"instance_id":         "id1",
				"instance_index":      0,
			})
			merge(attributes1, map[string]interface{}{
				"application_id":      "1",
				"application_version": "def",
				"instance_id":         "id2",
				"instance_index":      1,
			})
		})
		JustBeforeEach(func() {
			instance.SetState(dea.STATE_RUNNING)
			instance_registry.Register(instance)
			instance_registry.Register(instance1)
		})

		filtered_instances := func(message map[string]interface{}) []dea.Instance {
			instances := make([]dea.Instance, 0, 2)
			instance_registry.Instances_filtered_by_message(message, func(i dea.Instance) {
				instances = append(instances, i)
			})

			return instances
		}

		Context("when the app id doesn't match anything", func() {
			It("does not yield anything", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": ""})).To(HaveLen(0))
			})
		})

		Context("when the app id matches some instances", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1"})).To(Equal([]dea.Instance{instance, instance1}))
			})
		})

		Context("when filtered by version", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1", "version": "abc"})).To(Equal([]dea.Instance{instance}))
			})
		})

		Context("when filtered by instances", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1", "instances": []string{"id2"}})).To(Equal([]dea.Instance{instance1}))
			})
		})

		Context("when filtered by instance_ids", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1", "instance_ids": []string{"id2"}})).To(Equal([]dea.Instance{instance1}))
			})
		})

		Context("when filtered by indices", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1", "indices": []int{0}})).To(Equal([]dea.Instance{instance}))
			})
		})

		Context("when filtered by state", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1", "states": []string{"RUNNING", "STARTING"}})).To(Equal([]dea.Instance{instance}))
			})
		})

		Context("when filtered by version, instances, indices, state", func() {
			It("returns matching instances of the app", func() {
				Expect(filtered_instances(map[string]interface{}{"droplet": "1",
					"version":   "abc",
					"instances": []string{"id1"},
					"indices":   []int{0, 1},
					"states":    []string{"RUNNING", "BORN"},
				})).To(Equal([]dea.Instance{instance}))
			})
		})

	})

	Describe("crash reaping of orphans", func() {
		var tmpdir string

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance_registry")
			config.BaseDir = tmpdir
			config.CrashesPath = tmpdir
		})
		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("should reap orphaned crashes", func() {
			instance = register_crashed_instance(nil, nil)
			instance_registry.reapOrphanedCrashes()

			Expect(is_reaped(instance)).To(BeTrue())
		})

		It("should ignore referenced crashes", func() {
			instance = register_crashed_instance(instance_registry, nil)
			instance_registry.reapOrphanedCrashes()

			Expect(is_reaped(instance)).To(BeFalse())
		})
	})

	Describe("crash reaping", func() {
		var tmpdir string
		crash_lifetime := 10 * time.Second

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance_registry")
			config.BaseDir = tmpdir
			config.CrashesPath = tmpdir
			config.CrashLifetime = crash_lifetime
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("should reap crashes that are too old", func() {
			now := time.Now()
			instance1 = register_crashed_instance(instance_registry, map[string]interface{}{
				"application_id": "1",
			})
			instance1.state_times[instance1.State()] = now.Add(5 * time.Second)

			instance2 := register_crashed_instance(instance_registry, map[string]interface{}{
				"application_id": "2",
			})
			instance2.state_times[instance2.State()] = now.Add(-15 * time.Second)

			instance_registry.reapCrashes()

			Expect(is_reaped(instance1)).To(BeFalse())
			Expect(is_reaped(instance2)).To(BeTrue())
		})

		It("should reap all but the most recent crash for an app", func() {
			now := time.Now()
			instance1 := register_crashed_instance(instance_registry, nil)
			instance2 := register_crashed_instance(instance_registry, nil)
			instance2.state_times[instance2.State()] = now.Add(-1 * time.Second)
			instance3 := register_crashed_instance(instance_registry, nil)
			instance3.state_times[instance3.State()] = now.Add(-2 * time.Second)

			instance_registry.reapCrashes()

			Expect(is_reaped(instance1)).To(BeFalse())
			Expect(is_reaped(instance2)).To(BeTrue())
			Expect(is_reaped(instance3)).To(BeTrue())
		})
	})

	Describe("crash reaping under disk pressure", func() {
		var tmpdir string
		var statfsResponse []syscall.Statfs_t

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance_registry")
			config.BaseDir = tmpdir
			config.CrashesPath = tmpdir

			config.CrashBlockUsageRatioThreshold = 0.5
			config.CrashInodeUsageRatioThreshold = 0.5
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		JustBeforeEach(func() {
			instance_registry.statfs = func(path string, buf *syscall.Statfs_t) error {
				buf.Blocks = statfsResponse[0].Blocks
				buf.Bfree = statfsResponse[0].Bfree
				buf.Files = statfsResponse[0].Files
				buf.Ffree = statfsResponse[0].Ffree

				if len(statfsResponse) > 1 {
					statfsResponse = statfsResponse[1:]
				}
				return nil
			}
		})

		It("should reap under disk pressure", func() {
			statfsResponse = []syscall.Statfs_t{
				{Blocks: 1, Bfree: 0, Files: 1, Ffree: 0}, //true
				{Blocks: 1, Bfree: 1, Files: 1, Ffree: 1}, //false
			}

			instance1 := register_crashed_instance(instance_registry, nil)
			instance2 := register_crashed_instance(instance_registry, nil)
			instance2.state_times[instance2.State()] = time.Now().Add(1 * time.Second)

			instance_registry.reapCrashesUnderDiskPressure()

			Expect(is_reaped(instance1)).To(BeTrue())
			Expect(is_reaped(instance2)).To(BeFalse())

		})

		It("should continue reaping while under disk pressure", func() {
			statfsResponse = []syscall.Statfs_t{
				{Blocks: 1, Bfree: 0, Files: 1, Ffree: 0}, //true
			}

			instance1 := register_crashed_instance(instance_registry, nil)
			instance2 := register_crashed_instance(instance_registry, nil)
			instance2.state_times[instance2.State()] = time.Now().Add(1 * time.Second)

			instance_registry.reapCrashesUnderDiskPressure()

			Expect(is_reaped(instance1)).To(BeTrue())
			Expect(is_reaped(instance2)).To(BeTrue())
		})
	})

	Describe("reap_crash", func() {
		var tmpdir string

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance_registry")
			config.BaseDir = tmpdir
			config.CrashesPath = tmpdir
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("logs to the loggregator", func() {
			fakeEmitter := temitter.FakeEmitter{}
			loggregator.SetEmitter(&fakeEmitter)

			instance_registry.Register(instance)
			instance_registry.reapCrash(instance.instance_id, "no reason", func() {
				instance_registry.reapCrashesUnderDiskPressure()
			})

			Expect(fakeEmitter.Messages).To(HaveLen(1))
			Expect(fakeEmitter.Messages[instance.ApplicationId()][1]).To(Equal("Removing crash for app with id 37"))
		})
	})

	Describe("hasDiskPressure", func() {
		var tmpdir string
		var statfsResponse []syscall.Statfs_t

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance_registry")
			config.BaseDir = tmpdir
			config.CrashesPath = tmpdir

			config.CrashBlockUsageRatioThreshold = 0.5
			config.CrashInodeUsageRatioThreshold = 0.5

		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		JustBeforeEach(func() {
			instance_registry.statfs = func(path string, buf *syscall.Statfs_t) error {
				*buf = statfsResponse[0]

				if len(statfsResponse) > 1 {
					statfsResponse = statfsResponse[1:]
				}
				return nil
			}
		})

		It("should return false when #stat raises", func() {
			instance_registry.statfs = func(path string, buf *syscall.Statfs_t) error {
				return errors.New("error")
			}

			Expect(instance_registry.hasDiskPressure()).To(BeFalse())
		})

		It("should return false when thresholds are not reached", func() {
			statfsResponse = []syscall.Statfs_t{
				{Blocks: 10, Bfree: 8, Files: 10, Ffree: 8},
			}

			Expect(instance_registry.hasDiskPressure()).To(BeFalse())
		})

		It("should return true when block threshold is reached", func() {
			statfsResponse = []syscall.Statfs_t{
				{Blocks: 10, Bfree: 2, Files: 10, Ffree: 8},
			}

			Expect(instance_registry.hasDiskPressure()).To(BeTrue())
		})

		It("should return true when inode threshold is reached", func() {
			statfsResponse = []syscall.Statfs_t{
				{Blocks: 10, Bfree: 8, Files: 10, Ffree: 2},
			}

			Expect(instance_registry.hasDiskPressure()).To(BeTrue())
		})
	})
})
