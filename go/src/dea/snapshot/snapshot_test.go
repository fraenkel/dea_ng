package snapshot_test

import (
	cfg "dea/config"
	"dea/droplet"
	. "dea/snapshot"
	"dea/staging"
	"dea/starting"
	thelpers "dea/testhelpers"
	tboot "dea/testhelpers/boot"
	tstaging "dea/testhelpers/staging"
	tstarting "dea/testhelpers/starting"
	"dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path"
	"time"
)

var _ = Describe("Snapshot", func() {
	var config cfg.Config
	var snapshot Snapshot
	var ir starting.InstanceRegistry
	var str *staging.StagingTaskRegistry
	var dropletRegistry droplet.DropletRegistry
	var im InstanceManager
	var baseDir string
	var instances []*starting.Instance

	registerInstance := func(state starting.State) *starting.Instance {
		attributes := thelpers.Valid_instance_attributes(true)
		instance := starting.NewInstance(attributes, &config, dropletRegistry, "127.0.0.1")
		instance.SetState(state)
		ir.Register(instance)

		return instance
	}

	registerStagingTask := func() *tstaging.FakeStagingTask {
		taskAttrs := thelpers.Valid_staging_attributes()
		task := &tstaging.FakeStagingTask{
			StagingMsg: staging.NewStagingMessage(taskAttrs),
		}

		str.Register(task)
		return task
	}

	BeforeEach(func() {
		baseDir, _ = ioutil.TempDir("", "snapshot")
		os.MkdirAll(path.Join(baseDir, "tmp"), 0755)
		os.MkdirAll(path.Join(baseDir, "db"), 0755)

		str = staging.NewStagingTaskRegistry(nil)

		registerStagingTask()
		registerStagingTask()

		ir = tstarting.NewFakeInstanceRegistry()
		instances = make([]*starting.Instance, 0, len(starting.STATES))
		for _, s := range starting.STATES {
			instances = append(instances, registerInstance(s))
		}
	})
	AfterEach(func() {
		os.RemoveAll(baseDir)
	})

	JustBeforeEach(func() {
		snapshot = NewSnapshot(ir, str, im, baseDir)
	})

	Describe("save", func() {
		It("saves the timestamp to the snapshot", func() {
			snapshot.Save()

			saved := Snap{}
			err := utils.Yaml_Load(snapshot.Path(), &saved)
			Expect(err).To(BeNil())
			Expect(saved.Time).To(BeNumerically("~", int(time.Now().UnixNano()), 50*time.Millisecond))
		})

		It("saves the instance registry", func() {
			snapshot.Save()

			saved := Snap{}
			utils.Yaml_Load(snapshot.Path(), &saved)

			actual := make(map[string]bool)
			for _, i := range saved.Instances {
				actual[i["state"].(string)] = true
			}

			Expect(actual).To(HaveLen(2))
			Expect(actual["running"]).To(BeTrue())
			Expect(actual["crashed"]).To(BeTrue())
		})

		It("saves the staging task registry", func() {
			snapshot.Save()

			saved := Snap{}
			utils.Yaml_Load(snapshot.Path(), &saved)

			Expect(saved.Staging_tasks).To(HaveLen(2))

			tasks := str.Tasks()
			Expect(saved.Staging_tasks[0]["task_id"]).To(Equal(tasks[0].Id()))
			Expect(saved.Staging_tasks[1]["task_id"]).To(Equal(tasks[1].Id()))
		})
	})

	Context("instances fields", func() {
		var instance map[string]interface{}
		JustBeforeEach(func() {
			snapshot.Save()

			saved := Snap{}
			utils.Yaml_Load(snapshot.Path(), &saved)
			Expect(saved.Time).To(BeNumerically("~", int(time.Now().UnixNano()), 50*time.Millisecond))
			instance = saved.Instances[0]
		})

		It("has a snapshot with expected attributes so loggregator can process the json correctly", func() {
			keys := []string{
				"application_id",
				"warden_container_path",
				"warden_job_id",
				"instance_index",
				"state",
				"syslog_drain_urls",
			}

			for _, k := range keys {
				Expect(instance).To(HaveKey(k))
			}
		})

		It("has extra keys for debugging purpose", func() {
			keys := []string{
				"warden_host_ip",
				"instance_host_port",
				"instance_id",
			}

			for _, k := range keys {
				Expect(instance).To(HaveKey(k))
			}
		})

		It("has correct drain urls", func() {
			drains := instance["syslog_drain_urls"].([]interface{})
			Expect(drains).To(HaveLen(2))

			expected := []string{"syslog://log.example.com", "syslog://log2.example.com"}
			for i, d := range drains {
				Expect(d).Should(Equal(expected[i]))
			}
		})

		It("has the instance's start timestamp so we can report its uptime", func() {
			Expect(instance).To(HaveKey("state_starting_timestamp"))
		})
	})

	Describe("load", func() {
		var fakeim tboot.FakeInstanceManager

		BeforeEach(func() {
			im = &fakeim
		})

		JustBeforeEach(func() {
			snap := Snap{Time: time.Now().UnixNano(),
				Instances: []map[string]interface{}{
					instances[0].Snapshot_attributes(),
					instances[1].Snapshot_attributes(),
				}}

			bytes, _ := goyaml.Marshal(snap)
			ioutil.WriteFile(snapshot.Path(), bytes, 0755)
		})

		It("should load a snapshot", func() {
			instances = make([]*starting.Instance, 0, 2)
			resume := 0
			born := 0
			start := 0

			fakeim.CreateCallback = func(attributes map[string]interface{}) *starting.Instance {
				Expect(attributes).ToNot(HaveKey("state"))
				i := starting.NewInstance(attributes, &config, dropletRegistry, "127.0.0.1")
				i.On(starting.Transition{starting.STATE_BORN, starting.STATE_RESUMING}, func() {
					resume++
				})
				i.On(starting.Transition{starting.STATE_RESUMING, starting.STATE_BORN}, func() {
					born++
				})
				i.On(starting.Transition{starting.STATE_RESUMING, starting.STATE_STARTING}, func() {
					start++
				})

				instances = append(instances, i)
				return i
			}

			snapshot.Load()
			Expect(instances).To(HaveLen(2))
			Expect(resume).To(Equal(2))
			Expect(born).To(Equal(1))
			Expect(start).To(Equal(1))
		})
	})
})
