package boot

import (
	cfg "dea/config"
	"dea/protocol"
	"dea/responders"
	"dea/staging"
	"dea/starting"
	thelpers "dea/testhelpers"
	tboot "dea/testhelpers/boot"
	tlogger "dea/testhelpers/logger"
	trm "dea/testhelpers/resource_manager"
	tstaging "dea/testhelpers/staging"
	tstarting "dea/testhelpers/starting"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/gorouter/common"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"syscall"
	"time"
)

var _ = Describe("Boot", func() {
	var config cfg.Config
	var tmpdir string
	var boot *bootstrap
	var fakenats *fakeyagnats.FakeYagnats

	registerInstance := func(m map[string]interface{}) *starting.Instance {
		attributes := thelpers.Valid_instance_attributes(false)
		for k, v := range m {
			attributes[k] = v
		}
		instance := starting.NewInstance(attributes, &config, boot.dropletRegistry, "127.0.0.1")
		instance.Task.TaskPromises = &fakePromises{}
		boot.instanceRegistry.Register(instance)
		return instance
	}

	registerStagingTask := func() *tstaging.FakeStagingTask {
		taskAttrs := thelpers.Valid_staging_attributes()
		task := &tstaging.FakeStagingTask{
			StagingMsg: staging.NewStagingMessage(taskAttrs),
		}

		boot.stagingTaskRegistry.Register(task)
		return task
	}

	setupNats := func() {
		boot.setupNats()
		fakenats = fakeyagnats.New()
		boot.nats.NatsClient = fakenats
	}

	setupDirectoryServers := func() {
		boot.localIp = "127.0.0.1"
		boot.setupDirectoryServers()
	}

	BeforeEach(func() {
		tmpdir, _ = ioutil.TempDir("", "main")
		config, _ = cfg.NewConfig(func(c *cfg.Config) error {
			c.BaseDir = tmpdir
			c.DirectoryServer = cfg.DirServerConfig{
				V1Port: 12345,
				V2Port: 23456,
			}
			c.Domain = "default"
			return nil
		})

	})

	JustBeforeEach(func() {
		boot = newBootstrap(&config)
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	Describe("logging setup", func() {
		It("should use a file sink when specified", func() {
			config.Logging.File = path.Join(tmpdir, "out.log")
			logger, err := boot.setupLogger()
			Expect(err).To(BeNil())

			logger.Info("test123")

			bytes, err := ioutil.ReadFile(config.Logging.File)
			Expect(string(bytes)).To(ContainSubstring(`"message":"test123"`))
		})

		It("should set the default log level when specified", func() {
			config.Logging.File = path.Join(tmpdir, "out.log")
			config.Logging.Level = "debug"
			logger, _ := boot.setupLogger()
			logger.Debug("debug123")

			bytes, _ := ioutil.ReadFile(config.Logging.File)
			Expect(string(bytes)).To(ContainSubstring(`"log_level":"debug"`))
		})

		It("logs the creation of the DEA", func() {
			config.Logging.File = path.Join(tmpdir, "out.log")
			boot.setupLogger()

			bytes, _ := ioutil.ReadFile(config.Logging.File)
			Expect(string(bytes)).To(ContainSubstring(`Dea started"`))
		})
	})

	Describe("loggregator setup", func() {

		It("should configure when router is valid", func() {
			config.Index = 0
			config.Loggregator = cfg.LoggregatorConfig{
				Router:       "localhost:5432",
				SharedSecret: "secret",
			}

			//      LoggregatorEmitter::Emitter.should_receive(:new).with("localhost:5432", "DEA", 0, "secret")
			//      LoggregatorEmitter::Emitter.should_receive(:new).with("localhost:5432", "STG", 0, "secret")
			boot.setupLoggregator()
		})

		It("should validate host", func() {
			config.Index = 0
			config.Loggregator = cfg.LoggregatorConfig{
				Router:       "null:5432",
				SharedSecret: "secret",
			}

			err := boot.setupLoggregator()
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("signal handlers", func() {
		Context("", func() {
			var c chan os.Signal
			JustBeforeEach(func() {
				c = make(chan os.Signal, 1)
				boot.signalHandler = &fakeSignalHandler{c}
				boot.setupSignalHandlers()
			})

			AfterEach(func() {
				boot.ignoreSignals()
			})

			test_signal := func(s syscall.Signal) {
				It("should trap "+s.String(), func() {
					syscall.Kill(syscall.Getpid(), s)
					rs := <-c
					Expect(rs).To(Equal(s))
				})
			}

			test_signal(syscall.SIGTERM)
			test_signal(syscall.SIGINT)
			test_signal(syscall.SIGTERM)
			test_signal(syscall.SIGQUIT)
			test_signal(syscall.SIGUSR1)
			test_signal(syscall.SIGUSR2)
		})

		Describe("handling USR1", func() {
			JustBeforeEach(func() {
				boot.setupRegistries()
				setupNats()

				boot.component = &common.VcapComponent{UUID: "bogus"}

				boot.setupSignalHandlers()
			})

			AfterEach(func() {
				boot.ignoreSignals()
			})

			It("sends dea.shutdown", func() {
				syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
				time.Sleep(100 * time.Millisecond)
				Expect(fakenats.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
			})

			It("stops any responder", func() {
				fakeresponder := &fakeResponder{}
				boot.responders = []responders.Responder{fakeresponder}
				syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
				time.Sleep(100 * time.Millisecond)
				Expect(fakeresponder.stopCalled).To(BeTrue())
			})
		})
	})

	Describe("directory setup", func() {
		JustBeforeEach(func() {
			boot.setupDirectories()
		})

		dirExists := func(dir string) {
			It("should create "+dir, func() {
				Expect(utils.File_Exists(path.Join(tmpdir, dir))).To(BeTrue())
			})
		}

		dirExists("db")
		dirExists("droplets")
		dirExists("instances")
		dirExists("tmp")
		dirExists("staging")
	})

	Describe("pid file setup", func() {
		It("should create a pid file", func() {
			pidFilename := path.Join(tmpdir, "pid")
			config.PidFile = pidFilename

			err := boot.setupPidFile()
			Expect(err).To(BeNil())
			Expect(utils.File_Exists(pidFilename)).To(BeTrue())
		})

		It("should return an error when it can't create the pid file", func() {
			pidFilename := path.Join(tmpdir, "doesnt_exist", "pid")
			config.PidFile = pidFilename

			err := boot.setupPidFile()
			Expect(err).ToNot(BeNil())
		})
	})

	Describe("shutdown", func() {
		JustBeforeEach(func() {
			boot.Terminator = &fakeTerminator{nested: boot}
			boot.setupLogger()
			boot.setupRegistries()
			setupNats()
			setupDirectoryServers()
			boot.component = &common.VcapComponent{UUID: "bogus"}
			boot.setupRouterClient()
		})

		It("sends a dea shutdown message", func() {
			boot.Shutdown()
			Expect(fakenats.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
		})

		Context("when instances are registered", func() {
			JustBeforeEach(func() {
				instance := registerInstance(nil)
				instance.SetState(starting.STATE_RUNNING)
				boot.instanceRegistry.Register(instance)
			})

			It("stops registered instances", func() {
				boot.Shutdown()
				instances := boot.instanceRegistry.Instances()
				Expect(instances).To(HaveLen(1))
				for _, i := range instances {
					Expect(i.State()).To(Equal(starting.STATE_STOPPED))
				}
			})
		})

		Context("when staging tasks are registered", func() {
			var task *tstaging.FakeStagingTask

			JustBeforeEach(func() {
				task = registerStagingTask()
			})

			It("stops registered instances", func() {
				boot.Shutdown()
				Expect(task.Stopped).To(BeTrue())
			})
		})

		It("should stop and flush nats", func() {
			boot.Shutdown()

			Expect(fakenats.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
			Expect(fakenats.PublishedMessages["router.unregister"]).To(HaveLen(1))
			msg := fakenats.PublishedMessages["router.unregister"][0]
			data := make(map[string]interface{})
			json.Unmarshal(msg.Payload, &data)
			Expect(data["host"]).To(Equal(boot.localIp))
			Expect(data["port"]).To(Equal(float64(config.DirectoryServer.V2Port)))
			Expect(data["uris"]).To(HaveLen(1))
			uris := data["uris"].([]interface{})
			Expect(uris[0]).To(ContainSubstring(config.Domain))
		})
	})

	Describe("evacuation", func() {
		var faketerminator *fakeTerminator

		JustBeforeEach(func() {
			faketerminator = &fakeTerminator{nested: boot}
			boot.Terminator = faketerminator

			boot.setupLogger()
			boot.setupRegistries()
			setupDirectoryServers()

			setupNats()

			boot.component = &common.VcapComponent{UUID: "bogus"}
			boot.setupRouterClient()
		})

		It("sends a dea evacuation message", func() {
			boot.evacuate()
			Expect(fakenats.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
		})

		It("should send an exited message for each instance", func() {
			instance := registerInstance(nil)
			instance.SetState(starting.STATE_RUNNING)
			boot.instanceRegistry.Register(instance)

			boot.evacuate()
			Expect(fakenats.PublishedMessages["droplet.exited"]).To(HaveLen(1))
			data := make(map[string]interface{})
			json.Unmarshal(fakenats.PublishedMessages["droplet.exited"][0].Payload, &data)
			Expect(data["instance"]).To(Equal(instance.Id()))
		})

		It("should call shutdown after some time", func() {
			c := make(chan time.Time, 1)
			boot.config.EvacuationDelay = 200 * time.Millisecond
			faketerminator.shutdownCallback = func() {
				c <- time.Now()
				close(c)
			}

			start := time.Now()
			boot.evacuate()

			called := <-c
			Expect(called.Sub(start)).To(BeNumerically("~", boot.config.EvacuationDelay, 10*time.Millisecond))
		})
	})

	Describe("send_shutdown_message", func() {
		It("publishes a dea.shutdown message on NATS", func() {
			setupNats()

			boot.component = &common.VcapComponent{UUID: "bogus"}
			boot.localIp = "1.2.3.4"
			boot.setupRegistries()
			instance := registerInstance(nil)

			boot.sendShutdownMessage()
			data := make(map[string]interface{})
			json.Unmarshal(fakenats.PublishedMessages["dea.shutdown"][0].Payload, &data)
			expected := map[string]interface{}{
				"id":              boot.component.UUID,
				"ip":              boot.localIp,
				"version":         protocol.VERSION,
				"app_id_to_count": map[string]interface{}{instance.ApplicationId(): float64(1)},
			}
			Expect(data).To(Equal(expected))
		})
	})

	Describe("reap_unreferenced_droplets", func() {
		JustBeforeEach(func() {
			faketerminator := &fakeTerminator{nested: boot}
			boot.Terminator = faketerminator

			boot.setupLogger()
			boot.setupRegistries()

			for _, d := range []string{"a", "b", "c", "d", "e", "f"} {
				boot.dropletRegistry.Get(d)
			}

			registerInstance(map[string]interface{}{"droplet_sha1": "a"})
			registerInstance(map[string]interface{}{"droplet_sha1": "b"})

			t := registerStagingTask()
			t.Droplet_Sha1 = "e"
			t = registerStagingTask()
			t.Droplet_Sha1 = "f"
		})

		It("should delete any unreferenced droplets from the registry", func() {
			instances := boot.instanceRegistry.Instances()
			tasks := boot.stagingTaskRegistry.Tasks()
			expected := make([]string, 0, len(instances)+len(tasks))
			for _, i := range instances {
				expected = append(expected, i.DropletSHA1())
			}
			for _, t := range tasks {
				expected = append(expected, t.DropletSHA1())
			}

			boot.reapUnreferencedDroplets()
			shas := boot.dropletRegistry.SHA1s()
			Expect(shas).To(Equal(expected))
		})

		It("should destroy any unreferenced droplets", func() {
			droplet := boot.dropletRegistry.Get("a")
			boot.reapUnreferencedDroplets()

			// allow for background deletions to occur
			time.Sleep(10 * time.Millisecond)
			shas := boot.dropletRegistry.SHA1s()
			files, _ := ioutil.ReadDir(path.Dir(droplet.Dir()))
			actual := make([]string, 0, len(shas))
			for _, f := range files {
				actual = append(actual, f.Name())
			}
			Expect(actual).To(Equal(shas))
		})
	})

	Describe("setupVarz", func() {
		BeforeEach(func() {
			config.Stacks = []string{"Linux"}
		})

		It("adds stacks to varz", func() {
			setupNats()

			timer := boot.setupComponent()
			timer.Stop()

			m := boot.component.Varz.UniqueVarz.(map[string]interface{})
			Expect(m["stacks"]).To(Equal([]string{"Linux"}))
		})
	})

	Describe("periodic_varz_update", func() {
		var fakerm *trm.FakeResourceManager
		JustBeforeEach(func() {
			boot.setupRegistries()
			fakerm = &trm.FakeResourceManager{}
			boot.resource_manager = fakerm

			timer := boot.setupComponent()
			timer.Stop()
		})

		Describe("can_stage", func() {
			It("is 0 when there is not enough free memory or disk space", func() {
				fakerm.Reservable = 0

				boot.periodic_varz_update()

				m := boot.component.Varz.UniqueVarz.(map[string]interface{})
				Expect(m["can_stage"]).To(Equal(0))
			})

			It("is 1 when there is enough memory and disk space", func() {
				fakerm.Reservable = 3

				boot.periodic_varz_update()

				m := boot.component.Varz.UniqueVarz.(map[string]interface{})
				Expect(m["can_stage"]).To(Equal(1))
			})
		})

		Describe("reservable_stagers", func() {
			It("uses the value from resource_manager.number_reservable", func() {
				fakerm.Reservable = 456

				boot.periodic_varz_update()

				m := boot.component.Varz.UniqueVarz.(map[string]interface{})
				Expect(m["reservable_stagers"]).To(BeNumerically("==", 456))
			})
		})

		Describe("available_memory_ratio", func() {
			It("uses the value from resource_manager.available_memory_ratio", func() {
				fakerm.MemoryRatio = 0.5

				boot.periodic_varz_update()

				m := boot.component.Varz.UniqueVarz.(map[string]interface{})
				Expect(m["available_memory_ratio"]).To(BeNumerically("==", 0.5))
			})
		})

		Describe("available_disk_ratio", func() {
			It("uses the value from resource_manager.available_disk_ratio", func() {
				fakerm.DiskRatio = 0.75

				boot.periodic_varz_update()

				m := boot.component.Varz.UniqueVarz.(map[string]interface{})
				Expect(m["available_disk_ratio"]).To(BeNumerically("==", 0.75))
			})
		})

		Describe("instance_registry", func() {
			Context("when an empty registry", func() {
				It("is an empty hash", func() {
					boot.periodic_varz_update()

					m := boot.component.Varz.UniqueVarz.(map[string]interface{})
					Expect(m["instance_registry"]).To(HaveLen(0))
				})
			})

			Context("with a registry with an instance of an app", func() {
				var instance1 *starting.Instance

				JustBeforeEach(func() {
					instance1 = registerInstance(nil)
				})

				It("inlines the instance registry grouped by app ID", func() {
					boot.periodic_varz_update()

					m := boot.component.Varz.UniqueVarz.(map[string]interface{})
					Expect(m["instance_registry"]).To(HaveLen(1))
					is := m["instance_registry"].(map[string]map[string]interface{})
					Expect(is).To(HaveKey(instance1.ApplicationId()))
					appInstances := is[instance1.ApplicationId()]
					i := appInstances[instance1.Id()].(map[string]interface{})
					Expect(i["state"]).To(Equal(starting.STATE_BORN))
					Expect(i["state_timestamp"]).To(Equal(instance1.StateTimestamp().UnixNano()))
				})

				It("uses the values from stat_collector", func() {
					instance1.StatCollector = &tstarting.FakeStatCollector{
						Stats: starting.Stats{
							UsedMemory:   28 * cfg.Kibi,
							UsedDisk:     40,
							ComputedPCPU: 0.123,
						},
					}

					boot.periodic_varz_update()

					m := boot.component.Varz.UniqueVarz.(map[string]interface{})
					is := m["instance_registry"].(map[string]map[string]interface{})
					appInstances := is[instance1.ApplicationId()]
					i := appInstances[instance1.Id()].(map[string]interface{})
					Expect(i["used_memory_in_bytes"]).To(Equal(28 * cfg.Kibi))
					Expect(i["used_disk_in_bytes"]).To(Equal(cfg.Disk(40)))
					Expect(i["computed_pcpu"]).To(Equal(float32(0.123)))
				})
			})

			Context("with a registry containing two instances of one app", func() {
				var instance1 *starting.Instance
				var instance2 *starting.Instance

				JustBeforeEach(func() {
					instance1 = registerInstance(nil)
					instance2 = registerInstance(nil)
				})

				It("inlines the instance registry grouped by app ID", func() {
					boot.periodic_varz_update()

					m := boot.component.Varz.UniqueVarz.(map[string]interface{})
					is := m["instance_registry"].(map[string]map[string]interface{})
					instance1.ApplicationId()
					Expect(is).To(HaveKey(instance1.ApplicationId()))
					appInstances := is[instance1.ApplicationId()]

					Expect(appInstances).To(HaveLen(2))
					Expect(appInstances).To(HaveKey(instance1.Id()))
					Expect(appInstances).To(HaveKey(instance2.Id()))
				})
			})
		})
	})

	Describe("start_nats", func() {
		JustBeforeEach(func() {
			setupNats()
			timer := boot.setupComponent()
			timer.Stop()
		})

		It("starts nats", func() {
			boot.startNats()
			Expect(fakenats.ConnectedConnectionProvider).ToNot(BeNil())
		})

		findResponder := func(rtype string) bool {
			for _, r := range boot.responders {
				t := reflect.TypeOf(r).Elem()
				if t.PkgPath()+"/"+t.Name() == rtype {
					return true
				}
			}
			return false
		}

		It("sets up staging responder", func() {
			boot.startNats()

			Expect(boot.responders).To(HaveLen(3))
			Expect(findResponder("dea/responders/Staging")).To(BeTrue())
		})
		It("sets up dea locator responder", func() {
			boot.startNats()

			Expect(boot.responders).To(HaveLen(3))
			Expect(findResponder("dea/responders/DeaLocator")).To(BeTrue())
		})
		It("sets up staging locator responder", func() {
			boot.startNats()

			Expect(boot.responders).To(HaveLen(3))
			Expect(findResponder("dea/responders/StagingLocator")).To(BeTrue())
		})
	})

	Describe("start_finish", func() {
		JustBeforeEach(func() {
			boot.setupRegistries()
			setupDirectoryServers()
			boot.component = &common.VcapComponent{UUID: "bogus"}
			setupNats()
		})

		It("publishes dea.start", func() {
			boot.start_finish()
			Expect(fakenats.PublishedMessages["dea.start"]).To(HaveLen(1))
		})

		It("invokes LocatorResponder's Advertise", func() {
			fakeresponder := &FakeResponder{}
			boot.responders = []responders.Responder{fakeresponder}

			boot.start_finish()
			Expect(fakeresponder.Advertised).To(BeTrue())
		})

		Context("when recovering from snapshots", func() {

			JustBeforeEach(func() {
				boot.setupLogger()
				i := registerInstance(nil)
				i.SetState(starting.STATE_RUNNING)
				registerInstance(nil)
			})

			It("heartbeats its registry", func() {
				boot.start_finish()
				Expect(fakenats.PublishedMessages["dea.heartbeat"]).To(HaveLen(1))
			})
		})
	})

	Describe("evacuate", func() {
		JustBeforeEach(func() {
			boot.Terminator = &fakeTerminator{nested: boot}
			boot.setupRegistries()
			boot.setupLogger()
			setupDirectoryServers()
			boot.component = &common.VcapComponent{UUID: "bogus"}
			setupNats()
			boot.setupRouterClient()
		})

		It("stops dea advertising/locating", func() {
			fakeresponder := &FakeResponder{}
			boot.responders = []responders.Responder{fakeresponder}

			boot.evacuate()
			Expect(fakeresponder.Stopped).To(BeTrue())
		})

	})

	Describe("handle_dea_directed_start", func() {
		JustBeforeEach(func() {
			boot.setupLogger()
			fakel := tlogger.FakeL{}
			boot.logger.L = &fakel

			fakerm := &trm.FakeResourceManager{Reserve: true}
			boot.resource_manager = fakerm

			boot.instanceRegistry = &tstarting.FakeInstanceRegistry{}
			boot.setupInstanceManager()
			setupNats()
		})

		It("creates an instance", func() {
			attrs := thelpers.Valid_instance_attributes(false)
			bytes, _ := json.Marshal(attrs)
			msg := yagnats.Message{Payload: bytes}

			boot.HandleDeaDirectedStart(&msg)
			Expect(boot.instanceRegistry.Instances()).To(HaveLen(1))
		})

	})

	Describe("start", func() {
		var fakesnap tboot.FakeSnapshot
		JustBeforeEach(func() {
			fakesnap = tboot.FakeSnapshot{}
			boot.snapshot = &fakesnap
			boot.resource_manager = &trm.FakeResourceManager{}
			boot.instanceRegistry = &tstarting.FakeInstanceRegistry{}

			boot.setupLogger()
			setupNats()
			setupDirectoryServers()
			timer := boot.setupComponent()
			timer.Stop()
		})

		Describe("snapshot", func() {
			It("loads the snapshot on startup", func() {
				boot.Start()
				Expect(fakesnap.LoadInvoked).To(BeTrue())
			})
		})
	})

})

type fakeSignalHandler struct {
	c chan os.Signal
}

func (sh *fakeSignalHandler) handleSignal(s os.Signal) {
	sh.c <- s
}

type fakeResponder struct {
	stopCalled bool
}

func (fr *fakeResponder) Start() {
}
func (fr *fakeResponder) Stop() {
	fr.stopCalled = true
}

type fakeTerminator struct {
	nested    Terminator
	terminate bool

	shutdownCallback func()
}

func (ft *fakeTerminator) Shutdown() {
	if ft.shutdownCallback != nil {
		ft.shutdownCallback()
	}

	ft.nested.Shutdown()
}
func (ft *fakeTerminator) Terminate() {
	if ft.terminate {
		ft.nested.Terminate()
	}
}

type fakePromises struct {
}

func (fp *fakePromises) Promise_stop() error {
	return nil
}
func (fp *fakePromises) Promise_destroy() {
}

type FakeResponder struct {
	Stopped    bool
	Advertised bool
}

func (fr *FakeResponder) Start() {
}
func (fr *FakeResponder) Stop() {
	fr.Stopped = true
}
func (fr *FakeResponder) Advertise() {
	fr.Advertised = true
}
