package starting

import (
	"dea"
	cfg "dea/config"
	"dea/health_check"
	thelpers "dea/testhelpers"
	tcnr "dea/testhelpers/container"
	tdroplet "dea/testhelpers/droplet"
	tlogger "dea/testhelpers/logger"
	"encoding/json"
	"errors"
	"github.com/cloudfoundry/gordon"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"launchpad.net/goyaml"
	"net"
	"os"
	"path"
	"strconv"
	"time"
)

var _ = Describe("Instance", func() {
	var config cfg.Config
	var dropletRegistry *tdroplet.FakeDropletRegistry
	var attributes map[string]interface{}

	var instance *Instance

	BeforeEach(func() {
		config = cfg.Config{}
		attributes = thelpers.Valid_instance_attributes(false)
	})

	JustBeforeEach(func() {
		dropletRegistry = &tdroplet.FakeDropletRegistry{}
		instance = NewInstance(attributes, &config, dropletRegistry, "127.0.0.1")
	})

	Describe("default attributes", func() {
		It("defaults exit status to -1", func() {
			Expect(instance.ExitStatus()).To(Equal(int64(-1)))
		})
	})

	Describe("attributes from start message", func() {
		Describe("instance attributes", func() {
			BeforeEach(func() {
				attributes["instance_index"] = 38
			})
			It("id", func() {
				Expect(instance.Id()).ToNot(Equal(""))
			})

			It("id", func() {
				Expect(instance.Index()).To(Equal(38))
			})

		})

		Describe("application attributes", func() {
			Context("new application attributes", func() {
				BeforeEach(func() {
					attributes["application_id"] = "99"
					attributes["application_version"] = "some version"
					attributes["application_name"] = "my_application"
					attributes["application_uris"] = []string{"foo.com", "bar.com"}
				})

				It("application_id", func() {
					Expect(instance.ApplicationId()).To(Equal("99"))
				})
				It("application_version", func() {
					Expect(instance.ApplicationVersion()).To(Equal("some version"))
				})
				It("application_name", func() {
					Expect(instance.ApplicationName()).To(Equal("my_application"))
				})
				It("application_uris", func() {
					Expect(instance.ApplicationUris()).To(Equal([]string{"foo.com", "bar.com"}))
				})
			})

			Context("old application attributes", func() {
				BeforeEach(func() {
					delete(attributes, "application_id")
					delete(attributes, "application_name")
					delete(attributes, "application_version")
					delete(attributes, "application_uris")

					attributes["droplet"] = "99"
					attributes["version"] = "some version"
					attributes["name"] = "my_application"
					attributes["uris"] = []string{"foo.com", "bar.com"}
				})

				It("application_id", func() {
					Expect(instance.ApplicationId()).To(Equal("99"))
				})
				It("application_version", func() {
					Expect(instance.ApplicationVersion()).To(Equal("some version"))
				})
				It("application_name", func() {
					Expect(instance.ApplicationName()).To(Equal("my_application"))
				})
				It("application_uris", func() {
					Expect(instance.ApplicationUris()).To(Equal([]string{"foo.com", "bar.com"}))
				})
			})

		})

		Describe("droplet attributes", func() {
			Context("new droplet attributes", func() {
				BeforeEach(func() {
					attributes["droplet_sha1"] = "deadbeef"
					attributes["droplet_uri"] = "http://foo.com/file.ext"
				})

				It("droplet_sha1", func() {
					Expect(instance.DropletSHA1()).To(Equal("deadbeef"))
				})
				It("droplet_uri", func() {
					Expect(instance.DropletUri()).To(Equal("http://foo.com/file.ext"))
				})
			})
			Context("old droplet attributes", func() {
				BeforeEach(func() {
					delete(attributes, "droplet_sha1")
					delete(attributes, "droplet_uri")
					attributes["sha1"] = "deadbeef"
					attributes["executableUri"] = "http://foo.com/file.ext"

				})

				It("droplet_sha1", func() {
					Expect(instance.DropletSHA1()).To(Equal("deadbeef"))
				})
				It("droplet_uri", func() {
					Expect(instance.DropletUri()).To(Equal("http://foo.com/file.ext"))
				})
			})
		})

		Describe("start_command from message data", func() {
			BeforeEach(func() {
				attributes["start_command"] = "start command"
			})

			It("start_command", func() {
				Expect(instance.StartCommand()).To(Equal("start command"))
			})

			Context("when the value is nil", func() {
				BeforeEach(func() {
					attributes["start_command"] = nil
				})

				It("start_command", func() {
					Expect(instance.StartCommand()).To(Equal(""))
				})
			})

			Context("when the key is not present", func() {
				BeforeEach(func() {
					delete(attributes, "start_command")
				})
				It("start_command", func() {
					Expect(instance.StartCommand()).To(Equal(""))
				})
			})
		})

		Describe("other attributes", func() {
			limits := LimitsData{1, 2, 3}
			env := map[string]string{"FOO": "BAR", "BAR": "", "QUX": ""}
			services := []ServiceData{ServiceData{Name: "redis"}}

			BeforeEach(func() {
				attributes["limits"] = map[string]interface{}{"mem": 1, "disk": 2, "fds": 3}
				attributes["env"] = []string{"FOO=BAR", "BAR=", "QUX"}
				attributes["services"] = []map[string]interface{}{{"name": "redis", "type": "redis"}}
			})

			It("limits", func() {
				Expect(instance.Limits()).To(Equal(limits))
			})
			It("environment", func() {
				Expect(instance.Environment()).To(Equal(env))
			})
			It("services", func() {
				Expect(instance.Services()).To(Equal(services))
			})
		})
	})

	Describe("attributes from snapshot", func() {
		Describe("container attributes", func() {
			BeforeEach(func() {
				attributes["warden_handle"] = "abc"
				attributes["instance_host_port"] = 1234
				attributes["instance_container_port"] = 5678
			})

			It("warden_handle", func() {
				Expect(instance.Container.Handle()).To(Equal("abc"))
			})
			It("instance_host_port", func() {
				Expect(instance.HostPort()).To(Equal(uint32(1234)))
			})
			It("instance_container_port", func() {
				Expect(instance.ContainerPort()).To(Equal(uint32(5678)))
			})
		})
	})

	Describe("resource limits", func() {
		It("exports the memory limit in bytes", func() {
			Expect(instance.MemoryLimit()).To(Equal(512 * cfg.Mebi))
		})

		It("exports the disk limit in bytes", func() {
			Expect(instance.DiskLimit()).To(Equal(128 * cfg.MB))
		})

		It("exports the file descriptor limit", func() {
			Expect(instance.FileDescriptorLimit()).To(Equal(uint64(5000)))
		})
	})

	Describe("SetState", func() {
		It("should set state_timestamp when invoked", func() {
			old_timestamp := instance.StateTimestamp()
			time.Sleep(1 * time.Millisecond)
			instance.SetState(dea.STATE_RUNNING)
			Expect(instance.StateTimestamp().After(old_timestamp)).To(BeTrue())
		})
	})

	Describe("IsConsumingMemory", func() {
		memory := func(state dea.State, outcome bool) {
			instance.SetState(state)
			Expect(instance.IsConsumingMemory()).To(Equal(outcome))
		}

		It("consuming memory", func() {
			memory(dea.STATE_BORN, true)
			memory(dea.STATE_STARTING, true)
			memory(dea.STATE_RUNNING, true)
			memory(dea.STATE_STOPPING, true)
		})
		It("not consuming memory", func() {
			memory(dea.STATE_STOPPED, false)
			memory(dea.STATE_CRASHED, false)
			memory(dea.STATE_DELETED, false)
			memory(dea.STATE_RESUMING, false)
		})
	})

	Describe("IsConsumingDisk", func() {
		disk := func(state dea.State, outcome bool) {
			instance.SetState(state)
			Expect(instance.IsConsumingDisk()).To(Equal(outcome))
		}

		It("consuming disk", func() {
			disk(dea.STATE_BORN, true)
			disk(dea.STATE_STARTING, true)
			disk(dea.STATE_RUNNING, true)
			disk(dea.STATE_STOPPING, true)
			disk(dea.STATE_CRASHED, true)
		})
		It("not consuming disk", func() {
			disk(dea.STATE_STOPPED, false)
			disk(dea.STATE_DELETED, false)
			disk(dea.STATE_RESUMING, false)
		})
	})

	Describe("stat collector", func() {
		var collector fakeCollector
		JustBeforeEach(func() {
			collector = fakeCollector{}
			instance.StatCollector = &collector
			instance.setup_stat_collector()
		})

		Context("when paused", func() {
			startOn := func(state dea.State) {
				Expect(collector.started).To(BeFalse())
				instance.SetState(state)
				Expect(collector.started).To(BeFalse())
				instance.SetState(dea.STATE_RUNNING)
				Expect(collector.started).To(BeTrue())
			}

			It("starts when moving from resuming", func() {
				startOn(dea.STATE_RESUMING)
			})

			It("starts when moving from starting", func() {
				startOn(dea.STATE_STARTING)
			})
		})

		Context("when running", func() {
			stopOn := func(state dea.State) {
				Expect(collector.stopped).To(BeFalse())
				instance.SetState(dea.STATE_RUNNING)
				Expect(collector.stopped).To(BeFalse())
				instance.SetState(state)
				Expect(collector.stopped).To(BeTrue())
			}

			It("stops when moving to stopping", func() {
				stopOn(dea.STATE_STOPPING)
			})

			It("stops when moving to crashed", func() {
				stopOn(dea.STATE_CRASHED)
			})
		})
	})

	Describe("attributes_and_stats from stat collector", func() {
		var collector fakeCollector
		JustBeforeEach(func() {
			collector = fakeCollector{}
			instance.StatCollector = &collector
			instance.setup_stat_collector()
		})

		It("returns the used_memory_in_bytes stat in the attributes_and_stats hash", func() {
			collector.stats.UsedMemory = 28 * cfg.Mebi
			Expect(instance.Attributes_and_stats()["used_memory_in_bytes"]).To(Equal(28 * cfg.Mebi))
		})

		It("returns the used_disk_in_bytes stat in the attributes_and_stats hash", func() {
			collector.stats.UsedDisk = cfg.Disk(40)
			Expect(instance.Attributes_and_stats()["used_disk_in_bytes"]).To(Equal(cfg.Disk(40)))
		})

		It("returns the computed_pcpu stat in the attributes_and_stats hash", func() {
			collector.stats.ComputedPCPU = 0.123
			Expect(instance.Attributes_and_stats()["computed_pcpu"]).To(Equal(float32(0.123)))
		})
	})

	Describe("promise_health_check unit test", func() {
		var container tcnr.FakeContainer

		container_path := "fake/container/path"

		BeforeEach(func() {
			attributes["application_uris"] = []string{}
		})

		JustBeforeEach(func() {
			container = tcnr.FakeContainer{FHandle: "fake handle"}
			instance.Container = &container
		})

		AfterEach(func() {
			instance.cancel_health_check()
		})

		It("updates the path and host ip", func() {
			container.FUpdatePathAndIpPath = container_path
			container.FUpdatePathAndIpHostIp = "fancy ip"

			b, err := instance.Promise_health_check()
			Expect(err).To(BeNil())
			Expect(b).To(BeTrue())
			Expect(instance.Container.Path()).To(Equal(container_path))
			Expect(instance.Container.HostIp()).To(Equal("fancy ip"))
		})
	})

	Describe("promise_health_check", func() {
		var container tcnr.FakeContainer

		JustBeforeEach(func() {
			instance.Container = &container
		})

		AfterEach(func() {
			instance.cancel_health_check()
		})

		Describe("via state file", func() {
			var tmpdir string

			writeStateFile := func(state string) {
				statefile_path := container_relative_path(tmpdir, "statefile.json")
				bytes, _ := json.Marshal(map[string]string{"state": state})
				ioutil.WriteFile(statefile_path, bytes, 0755)

			}

			BeforeEach(func() {
				tmpdir, _ = ioutil.TempDir("", "instance")
				container.FUpdatePathAndIpPath = tmpdir
			})

			JustBeforeEach(func() {
				manifest_path := container_relative_path(tmpdir, "droplet.yaml")
				os.MkdirAll(path.Dir(manifest_path), 0755)
				bytes, _ := goyaml.Marshal(map[string]string{"state_file": "statefile.json"})
				ioutil.WriteFile(manifest_path, bytes, 0755)

				writeStateFile("RUNNING")
			})

			AfterEach(func() {
				os.RemoveAll(tmpdir)
			})

			It("using a state file", func() {
				instance.Promise_health_check()
				statefile := instance.healthCheck.(health_check.StateFileReady)
				Expect(statefile).ToNot(BeNil())
			})

			It("sets a timeout of 5 minutes", func() {
				now := time.Now()
				_, err := instance.Promise_health_check()
				Expect(err).To(BeNil())
				statefile := instance.healthCheck.(health_check.StateFileReady)
				Expect(statefile.End_time.Sub(now)).To(BeNumerically("~", 5*time.Minute, 1*time.Second))
			})

			It("can succeed", func() {
				b, _ := instance.Promise_health_check()
				Expect(b).To(BeTrue())
			})

			It("can fail", func() {
				manifest_path := container_relative_path(tmpdir, "droplet.yaml")
				bytes, _ := goyaml.Marshal(map[string]string{"state_file": "statefile.json"})
				ioutil.WriteFile(manifest_path, bytes, 0755)
				writeStateFile("CRASHED")

				b, _ := instance.Promise_health_check()
				Expect(b).To(BeFalse())
			})
		})

		Describe("when the application has URIs", func() {
			var listener net.Listener

			BeforeEach(func() {
				config.MaximumHealthCheckTimeout = 60 * time.Second
				attributes["application_uris"] = []string{"some-test-app.my-cloudfoundry.com"}

				listener, _ = net.Listen("tcp", ":0")
				_, port, _ := net.SplitHostPort(listener.Addr().String())
				p, _ := strconv.ParseUint(port, 10, 32)
				container.Setup("handle", 999, uint32(p))
			})

			AfterEach(func() {
				listener.Close()
			})

			It("using a port open", func() {
				instance.Promise_health_check()
				portopen := instance.healthCheck.(health_check.PortOpen)
				Expect(portopen).ToNot(BeNil())
			})

			It("defaults to 60 seconds timeout", func() {
				now := time.Now()
				_, err := instance.Promise_health_check()
				Expect(err).To(BeNil())
				portopen := instance.healthCheck.(health_check.PortOpen)
				Expect(portopen.End_time.Sub(now)).To(BeNumerically("~", 60*time.Second, 1*time.Second))
			})

			Context("configurable timeout", func() {
				BeforeEach(func() {
					config.MaximumHealthCheckTimeout = 100 * time.Second
				})

				It("is adjusted to 100 seconds", func() {
					now := time.Now()
					_, err := instance.Promise_health_check()
					Expect(err).To(BeNil())
					portopen := instance.healthCheck.(health_check.PortOpen)
					Expect(100*time.Second - portopen.End_time.Sub(now)).To(BeNumerically("<", 1*time.Second))
				})
			})

			It("succeeds when the port is open", func() {
				b, err := instance.Promise_health_check()
				Expect(err).To(BeNil())
				Expect(b).To(BeTrue())
			})

			It("fails when the port is not open", func() {
				instance.healthCheckTimeout = 1 * time.Millisecond
				listener.Close()
				b, err := instance.Promise_health_check()
				Expect(err).To(BeNil())
				Expect(b).To(BeFalse())
			})
		})

		Describe("when the application does not have any URIs", func() {
			BeforeEach(func() {
				attributes["application_uris"] = []string{}
			})

			It("succeeds when the port is open", func() {
				b, err := instance.Promise_health_check()
				Expect(err).To(BeNil())
				Expect(b).To(BeTrue())
			})
		})

		Context("when failing to check the health", func() {
			BeforeEach(func() {
				container.FUpdatePathAndIpError = errors.New("error")
			})

			It("returns the error", func() {
				b, err := instance.Promise_health_check()
				Expect(err).ToNot(BeNil())
				Expect(b).To(BeFalse())
			})

			It("should log the failure", func() {
				logger := &tlogger.FakeL{}
				instance.Logger.L = logger
				instance.Promise_health_check()
				Expect(logger.Logs["error"][0]).To(Equal("droplet.health-check.container-info-failed: error"))
			})
		})
	})

	Describe("start transition", func() {
		var tmpdir string
		var container tcnr.FakeContainer
		var fakepromises *fakePromises

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance")
			config.BaseDir = tmpdir
			attributes["application_uris"] = nil

			fakepromises = &fakePromises{healthcheckResponse: true}
		})

		JustBeforeEach(func() {
			container = tcnr.FakeContainer{}
			instance.Container = &container
			fakepromises.realPromises = instance.InstancePromises
			instance.InstancePromises = fakepromises
			dropletRegistry.Get(instance.DropletSHA1())
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		startInstance := func() error {
			var err error
			started := false

			instance.Start(func(e error) error {
				err = e
				started = true
				return nil
			})
			Eventually(func() bool { return started }).Should(BeTrue())
			return err
		}

		Describe("checking source state", func() {
			BeforeEach(func() {
				fakepromises.stateInvoke = true
			})

			It("passes when BORN", func() {
				Expect(instance.State()).To(Equal(dea.STATE_BORN))
				err := startInstance()
				Expect(err).To(BeNil())
			})

			failStart := func(s dea.State) {
				It("fails when "+string(s), func() {
					instance.state = s
					err := startInstance()
					Expect(err.Error()).To(ContainSubstring("transition"))
				})
			}

			failStart(dea.STATE_STARTING)
			failStart(dea.STATE_RUNNING)
			failStart(dea.STATE_STOPPING)
			failStart(dea.STATE_STOPPED)
			failStart(dea.STATE_CRASHED)
			failStart(dea.STATE_DELETED)
			failStart(dea.STATE_RESUMING)
		})

		Describe("downloading droplet", func() {
			BeforeEach(func() {
				fakepromises.dropletInvoke = true
			})
			It("succeeds when download succeeds", func() {
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))

			})

			It("fails when download fails", func() {
				msg := "download failed"
				dropletRegistry.Droplet = tdroplet.FakeDroplet{DownloadError: errors.New(msg)}

				err := startInstance()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(msg))
			})
		})

		Describe("creating warden container", func() {
			srcpath := "/var/src/"
			dstpath := "/var/dst"

			BeforeEach(func() {
				config.BindMounts = []map[string]string{{"src_path": srcpath, "dst_path": dstpath}}
			})
			It("succeeds when the call succeeds", func() {

				droplet := dropletRegistry.Droplet
				droplet_dir := droplet.Dir()
				mount1 := warden.CreateRequest_BindMount{SrcPath: &droplet_dir, DstPath: &droplet_dir}
				mount2 := warden.CreateRequest_BindMount{SrcPath: &srcpath, DstPath: &dstpath}
				expected_bind_mounts := []*warden.CreateRequest_BindMount{&mount1, &mount2}

				startInstance()
				Expect(instance.ExitDescription()).To(Equal(""))

				Expect(container.FCreateBindMounts).To(Equal(expected_bind_mounts))
				Expect(container.FCreateDiskLimit).To(Equal(uint64(instance.DiskLimit())))
				Expect(container.FCreateMemoryLimit).To(Equal(uint64(instance.MemoryLimit())))
				Expect(container.FCreateNetwork).To(Equal(true))
			})

			It("fails when the call fails", func() {
				msg := "promise warden call error for container creation"
				container.FCreateError = errors.New(msg)

				err := startInstance()
				Expect(err.Error()).To(Equal(msg))
			})

			It("saves the created container's handle on attributes", func() {
				container.FCreateHandle = "some-handle"
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.Container.Handle()).To(Equal("some-handle"))
			})
		})
		Describe("extracting the droplet", func() {
			BeforeEach(func() {
				fakepromises.extractDropletInvoke = true
			})

			It("should run tar", func() {
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))
				Expect(container.FRunScript).To(ContainSubstring("tar zxf"))
			})

			It("can fail by run failing", func() {
				msg := "droplet extraction failure"
				container.FRunScriptError = errors.New(msg)
				err := startInstance()
				Expect(err.Error()).To(Equal(msg))
			})
		})
		Describe("setting up environment", func() {
			BeforeEach(func() {
				fakepromises.setupEnvironmentInvoke = true
			})

			It("should create the app dir", func() {
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))
				Expect(container.FRunScript).To(ContainSubstring("mkdir -p home/vcap/app"))
			})
			It("should chown the app dir", func() {
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))
				Expect(container.FRunScript).To(ContainSubstring("chown vcap:vcap home/vcap/app"))
			})
			It("should symlink the app dir", func() {
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))
				Expect(container.FRunScript).To(ContainSubstring("ln -s home/vcap/app /app"))
			})

			It("can fail by run failing", func() {
				msg := "environment setup failure"
				container.FRunScriptError = errors.New(msg)
				err := startInstance()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(msg))
			})
		})
		Describe("hook scripts", func() {
			var tmpdir string

			BeforeEach(func() {
				tmpdir, _ = ioutil.TempDir("", "instance")
				os.MkdirAll(path.Join(tmpdir, "hooks"), 0755)
				fakepromises.execHookScriptInvoke = true
			})
			AfterEach(func() {
				os.RemoveAll(tmpdir)
			})

			It("executes the before start hook", func() {
				before_script := path.Join(tmpdir, "hooks", "before_start")
				instance.hooks = map[string]string{"before_start": before_script}
				ioutil.WriteFile(before_script, []byte("before start"), 0755)
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(container.FRunScript).To(ContainSubstring("before start"))
			})
			It("executes the after start hook", func() {
				after_script := path.Join(tmpdir, "hooks", "after_start")
				instance.hooks = map[string]string{"after_start": after_script}
				ioutil.WriteFile(after_script, []byte("after start"), 0755)
				err := startInstance()
				Expect(err).To(BeNil())
				Expect(container.FRunScript).To(ContainSubstring("after start"))
			})
		})
		Describe("promise_start", func() {
			BeforeEach(func() {
				container.FSpawnJobId = 37
				fakepromises.startInvoke = true
			})

			It("raises errors when the request fails", func() {
				msg := "can't start the application"
				container.FSpawnError = errors.New(msg)

				err := instance.Promise_start()
				Expect(err.Error()).To(Equal(msg))

				// Job ID should not be set
				Expect(instance.warden_job_id).To(Equal(uint32(0)))
			})

			Context("when there is a task info yaml in the droplet", func() {
				It("generates the correct script and calls promise spawn", func() {
					script := "fake_start_command.sh"
					instance.stagedInfo = map[string]interface{}{"start_command": script}
					instance.Promise_start()
					Expect(container.FSpawnScript).To(ContainSubstring(script))
					Expect(container.FSpawnFileDescriptorLimit).To(Equal(instance.FileDescriptorLimit()))
					Expect(container.FSpawnNProc).To(Equal(uint64(NPROC_LIMIT)))
				})
			})

			Context("when there is a custom start command set on the instance", func() {
				custom_command := "my_custom_start_command.sh"

				BeforeEach(func() {
					attributes["start_command"] = custom_command
				})
				It("and the buildpack does not provide a command", func() {
					instance.stagedInfo = map[string]interface{}{"start_command": ""}
					instance.Promise_start()
					Expect(container.FSpawnScript).To(ContainSubstring(custom_command))
					Expect(container.FSpawnFileDescriptorLimit).To(Equal(instance.FileDescriptorLimit()))
					Expect(container.FSpawnNProc).To(Equal(uint64(NPROC_LIMIT)))
				})
				It("and the buildpack provides one", func() {
					instance.stagedInfo = map[string]interface{}{"start_command": "foo"}
					instance.Promise_start()
					Expect(container.FSpawnScript).To(ContainSubstring(custom_command))
					Expect(container.FSpawnFileDescriptorLimit).To(Equal(instance.FileDescriptorLimit()))
					Expect(container.FSpawnNProc).To(Equal(uint64(NPROC_LIMIT)))
				})
			})

			Context("when there is a staged_info but it lacks a start_command and instance lacks a start command", func() {
				It("when there is a staged_info but it lacks a start_command and instance lacks a start command", func() {
					instance.stagedInfo = map[string]interface{}{"start_command": ""}
					err := instance.Promise_start()
					Expect(err.Error()).To(Equal("missing start command"))
				})
			})
		})

		Describe("checking application health", func() {
			It("transitions from starting to running if healthy", func() {
				fakepromises.stateInvoke = true

				err := startInstance()
				Expect(err).To(BeNil())
				Expect(instance.ExitDescription()).To(Equal(""))
				Expect(fakepromises.stateFrom).To(Equal([]dea.State{dea.STATE_STARTING}))
				Expect(fakepromises.stateTo).To(Equal(dea.STATE_RUNNING))
			})

			It("fails if the instance is unhealthy", func() {
				fakepromises.stateInvoke = true
				fakepromises.healthcheckResponse = false

				err := startInstance()
				Expect(err).ToNot(BeNil())
				Expect(instance.ExitDescription()).To(Equal("failed to start accepting connections"))
			})
		})

	})

	Describe("stop transition", func() {
		var tmpdir string
		var container tcnr.FakeContainer
		var fakepromises *fakePromises

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance")
			config.BaseDir = tmpdir
			attributes["application_uris"] = nil

			fakepromises = &fakePromises{healthcheckResponse: true}
		})

		JustBeforeEach(func() {
			container = tcnr.FakeContainer{}
			instance.Container = &container
			fakepromises.realPromises = instance.InstancePromises
			instance.InstancePromises = fakepromises
			dropletRegistry.Get(instance.DropletSHA1())
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		stopInstance := func() error {
			var err error
			stopped := false
			instance.Stop(func(e error) error {
				stopped = true
				err = e
				return nil
			})
			Eventually(func() bool { return stopped }).Should(BeTrue())

			return err
		}

		Describe("checking source state", func() {
			BeforeEach(func() {
				fakepromises.stateInvoke = true
			})

			It("passes when RUNNING", func() {
				instance.state = dea.STATE_RUNNING
				err := stopInstance()
				Expect(err).To(BeNil())
			})

			It("passes when STARTING", func() {
				instance.state = dea.STATE_STARTING
				err := stopInstance()
				Expect(err).To(BeNil())
			})

			failStart := func(s dea.State) {
				It("fails when "+string(s), func() {
					instance.state = s
					err := stopInstance()
					Expect(err.Error()).To(ContainSubstring("transition"))
				})
			}

			failStart(dea.STATE_BORN)
			failStart(dea.STATE_STOPPING)
			failStart(dea.STATE_STOPPED)
			failStart(dea.STATE_CRASHED)
			failStart(dea.STATE_DELETED)
			failStart(dea.STATE_RESUMING)
		})

		Describe("hook scripts", func() {
			var tmpdir string

			BeforeEach(func() {
				tmpdir, _ = ioutil.TempDir("", "instance")
				os.MkdirAll(path.Join(tmpdir, "hooks"), 0755)
				fakepromises.execHookScriptInvoke = true
			})

			JustBeforeEach(func() {
				instance.Environment()["A"] = "B"
			})

			AfterEach(func() {
				os.RemoveAll(tmpdir)
			})

			It("executes the before stop hook", func() {
				before_script := path.Join(tmpdir, "hooks", "before_stop")
				instance.hooks = map[string]string{"before_stop": before_script}
				ioutil.WriteFile(before_script, []byte("before stop"), 0755)
				err := stopInstance()
				Expect(err).To(BeNil())
				Expect(container.FRunScript).To(ContainSubstring("before stop"))
			})
			It("executes the after stop hook", func() {
				after_script := path.Join(tmpdir, "hooks", "after_stop")
				instance.hooks = map[string]string{"after_stop": after_script}
				ioutil.WriteFile(after_script, []byte("after stop"), 0755)
				err := stopInstance()
				Expect(err).To(BeNil())
				Expect(container.FRunScript).To(ContainSubstring("after stop"))
			})

			It("exports the variables in the hook files", func() {
				after_script := path.Join(tmpdir, "hooks", "after_stop")
				instance.hooks = map[string]string{"after_stop": after_script}
				ioutil.WriteFile(after_script, []byte("after stop"), 0755)
				err := stopInstance()
				Expect(err).To(BeNil())
				Expect(container.FRunScript).To(ContainSubstring(`export A="B"`))
			})
		})
	})

	Describe("promise_link", func() {
		status := uint32(42)
		var container tcnr.FakeContainer
		var infoResponse warden.InfoResponse
		var linkResponse warden.LinkResponse

		BeforeEach(func() {
			infoResponse = warden.InfoResponse{
				Events: []string{},
			}

			linkResponse = warden.LinkResponse{
				ExitStatus: &status,
				Info:       &infoResponse,
			}
		})

		JustBeforeEach(func() {
			container = tcnr.FakeContainer{FHandle: "handle"}
			instance.Container = &container

			container.FLinkResponse = &linkResponse
		})

		Describe("when the LinkRequest fails", func() {
			It("propagates the exception", func() {
				container.FLinkError = errors.New("Runtime error")
				ip := instance.InstancePromises.(*instancePromises)
				_, err := ip.Promise_link()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("Runtime error"))
			})
		})

		Describe("when the LinkRequest completes successfully", func() {

			It("executes a LinkRequest with the warden handle and job ID and returns response", func() {
				instance.warden_job_id = 1

				ip := instance.InstancePromises.(*instancePromises)
				rsp, err := ip.Promise_link()
				Expect(err).To(BeNil())
				Expect(rsp.GetExitStatus()).To(Equal(status))
				Expect(container.FLinkJobId).To(Equal(uint32(1)))
			})
		})
	})

	Describe("Link", func() {
		status := uint32(42)
		var container tcnr.FakeContainer
		var fakepromises *fakePromises

		var infoResponse *warden.InfoResponse
		var linkResponse *warden.LinkResponse

		it_links := func() error {
			var err error
			linked := false
			instance.Link(func(e error) error {
				err = e
				linked = true
				return e
			})
			Eventually(func() bool { return linked }).Should(BeTrue())
			return err
		}

		BeforeEach(func() {
			container = tcnr.FakeContainer{FHandle: "handle"}
			fakepromises = &fakePromises{healthcheckResponse: true, linkInvoke: true}

			infoResponse = &warden.InfoResponse{
				Events: []string{},
			}

			linkResponse = &warden.LinkResponse{
				ExitStatus: &status,
				Info:       infoResponse,
			}

		})

		JustBeforeEach(func() {

			instance.Container = &container

			fakepromises.realPromises = instance.InstancePromises
			instance.InstancePromises = fakepromises

			container.FLinkResponse = linkResponse

			instance.state = dea.STATE_RUNNING
		})

		It("is triggered link when transitioning from RESUMING", func() {
			instance.state = dea.STATE_RESUMING
			instance.setup_link()

			instance.SetState(dea.STATE_RUNNING)
			Eventually(func() bool { return fakepromises.linkInvoked }).Should(BeTrue())
		})

		Describe("when promise_link succeeds", func() {
			It("sets the exit status on the instance", func() {
				it_links()
				Expect(instance.ExitStatus()).To(Equal(int64(status)))
			})

			Context("when the container_info has an event", func() {
				BeforeEach(func() {
					infoResponse.Events = []string{"some weird thing happened"}
				})

				It("sets the exit_description to the text of the event", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("some weird thing happened"))
				})
			})

			Context("when the info_response is missing", func() {
				BeforeEach(func() {
					linkResponse.Info = nil
				})
				It("sets the exit_description to 'cannot be determined'", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("cannot be determined"))
				})
			})

			Context("when there is an info_response no usable information", func() {
				It("sets the exit_description to 'app instance exited'", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("app instance exited"))
				})
			})

		})

		Describe("when the promise_link fails", func() {
			Context("when link errors", func() {
				BeforeEach(func() {
					container.FLinkError = errors.New("error")
				})

				It("sets exit status of the instance to -1", func() {
					it_links()
					Expect(instance.ExitStatus()).To(BeNumerically("==", -1))
				})

				It("sets the exit_description to 'unknonw'", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("unknown"))
				})
			})

			Context("when link fails", func() {
				JustBeforeEach(func() {
					fakepromises.linkInvoke = true
					status := uint32(255)
					info := warden.InfoResponse{
						Events: []string{"out of memory"},
					}
					rsp := warden.LinkResponse{
						ExitStatus: &status,
						Info:       &info,
					}
					container.FLinkResponse = &rsp
				})

				It("sets exit description based on link response", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("out of memory"))
				})
			})

			Context("when an arbitrary error occurs", func() {
				JustBeforeEach(func() {
					fakepromises.linkPanic = true
				})

				It("sets a generic exit description", func() {
					it_links()
					Expect(instance.ExitDescription()).To(Equal("unknown"))
				})
			})

		})

		Describe("state transitions", func() {
			It("changes to CRASHED when it was STARTING", func() {
				instance.state = dea.STATE_STARTING
				it_links()
				Expect(instance.state).To(Equal(dea.STATE_CRASHED))
			})

			It("changes to CRASHED when it was RUNNING", func() {
				instance.state = dea.STATE_RUNNING
				it_links()
				Expect(instance.state).To(Equal(dea.STATE_CRASHED))
			})

			It("doesn't changed when it was STOPPING", func() {
				instance.state = dea.STATE_STOPPING
				it_links()
				Expect(instance.state).To(Equal(dea.STATE_STOPPING))
			})

			It("doesn't changed when it was STOPPED", func() {
				instance.state = dea.STATE_STOPPED
				it_links()
				Expect(instance.state).To(Equal(dea.STATE_STOPPED))
			})
		})

	})

	Describe("destroy", func() {
		var container tcnr.FakeContainer

		BeforeEach(func() {
			container = tcnr.FakeContainer{FHandle: "handle"}
		})

		JustBeforeEach(func() {
			instance.Container = &container
		})

		Describe("promise_destroy", func() {
			It("executes a DestroyRequest", func() {
				destroyed := false
				instance.Destroy(func(e error) error {
					destroyed = true
					return nil
				})
				Eventually(func() bool { return destroyed }).Should(BeTrue())
				Expect(container.FDestroyInvoked).To(BeTrue())
			})
		})
	})

	Describe("health checks", func() {
		var tmpdir string
		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance")
		})
		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})
		Describe("promise_read_instance_manifest", func() {
			It("delivers {} if no container path is returned", func() {
				m, err := instance.Promise_read_instance_manifest("")
				Expect(err).To(BeNil())
				Expect(m).To(Equal(map[string]interface{}{}))
			})
			It("delivers {} if the manifest path doesn't exist", func() {
				m, err := instance.Promise_read_instance_manifest(tmpdir)
				Expect(err).To(BeNil())
				Expect(m).To(Equal(map[string]interface{}{}))
			})
			It("delivers the parsed manifest if the path exists", func() {
				manifest := map[string]interface{}{"test": "manifest"}
				bytes, _ := goyaml.Marshal(manifest)
				mpath := path.Join(tmpdir, "tmp", "rootfs", "home", "vcap", "droplet.yaml")
				os.MkdirAll(path.Dir(mpath), 0755)
				ioutil.WriteFile(mpath, bytes, 0755)

				m, err := instance.Promise_read_instance_manifest(tmpdir)
				Expect(err).To(BeNil())
				Expect(m).To(Equal(manifest))
			})
		})
	})

	Describe("crash handler", func() {
		var container tcnr.FakeContainer
		var fakepromises *fakePromises
		var tmpdir string

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance")
			config.CrashesPath = tmpdir
			container = tcnr.FakeContainer{FHandle: "fake handle"}
			fakepromises = &fakePromises{healthcheckResponse: true, crashHandlerInvoke: true}
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		JustBeforeEach(func() {

			instance.Container = &container

			fakepromises.realPromises = instance.InstancePromises
			instance.InstancePromises = fakepromises
			instance.Task.TaskPromises = fakepromises

			instance.setup_crash_handler()
			instance.state = dea.STATE_RUNNING
		})

		It("is triggered link when transitioning from RESUMING", func() {
			instance.state = dea.STATE_RESUMING
			instance.SetState(dea.STATE_CRASHED)
			Eventually(func() bool { return fakepromises.crashHandlerInvoked }).Should(BeTrue())
		})

		It("is triggered link when transitioning from RUNNING", func() {
			instance.state = dea.STATE_RUNNING
			instance.SetState(dea.STATE_CRASHED)
			Eventually(func() bool { return fakepromises.crashHandlerInvoked }).Should(BeTrue())
		})

		Describe("when triggered", func() {
			BeforeEach(func() {
				attributes["warden_handle"] = "handle"
			})

			crash_handler := func() {
				crashed := false
				instance.crash_handler(func(e error) error {
					crashed = true
					return e
				})
				Eventually(func() bool { return crashed }).Should(BeTrue())
			}

			It("should invoke promise_copy_out", func() {
				crash_handler()
				Expect(fakepromises.copyOutInvoked).To(BeTrue())
			})

			It("should invoke promise_destroy", func() {
				crash_handler()
				Expect(fakepromises.destroyInvoked).To(BeTrue())
			})

			It("should close warden connections", func() {
				crash_handler()
				Expect(container.FCloseAllConnectionsInvoked).To(BeTrue())
			})
		})

		Describe("promise_copy_out", func() {
			BeforeEach(func() {
				fakepromises.copyOutInvoke = true
			})

			It("should copy the contents of a directory", func() {
				instance.Promise_copy_out()
				Expect(container.FCopyOutSrc).To(Equal("/home/vcap/"))
				Expect(container.FCopyOutDest).To(ContainSubstring(tmpdir))
			})
		})
	})

	Describe("staged_info", func() {
		var container tcnr.FakeContainer
		var fakepromises *fakePromises
		var tmpdir string

		BeforeEach(func() {
			tmpdir, _ = ioutil.TempDir("", "instance")
			config.CrashesPath = tmpdir
			container = tcnr.FakeContainer{FHandle: "fake handle"}
			fakepromises = &fakePromises{healthcheckResponse: true, crashHandlerInvoke: true}
		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		JustBeforeEach(func() {
			instance.Container = &container

			fakepromises.realPromises = instance.InstancePromises
			instance.InstancePromises = fakepromises
			instance.Task.TaskPromises = fakepromises
		})

		Context("when the files does exist", func() {
			BeforeEach(func() {
				container.FCopyOutCallback = func(dest string) {
					manifest := map[string]interface{}{"a": 1}
					bytes, _ := goyaml.Marshal(manifest)
					os.MkdirAll(dest, 0755)
					ioutil.WriteFile(path.Join(dest, "staging_info.yml"), bytes, 0755)
				}
			})

			It("sends copying out request", func() {
				instance.staged_info()
				Expect(container.FCopyOutSrc).To(Equal("/home/vcap/staging_info.yml"))
			})

			It("reads the file from the copy out", func() {
				m := instance.staged_info()
				Expect(m).To(Equal(map[string]interface{}{"a": 1}))
			})

			It("should only be called once", func() {
				instance.staged_info()
				instance.staged_info()
				Expect(container.FCopyOutCount).To(Equal(1))
			})

		})

		Context("when the yaml file does not exist", func() {
			It("returns nil", func() {
				Expect(instance.staged_info()).To(Equal(map[string]interface{}(nil)))
			})
		})

		It("doesn't pollute the temp directory", func() {
			before, _ := ioutil.ReadDir(tmpdir)
			instance.staged_info()
			after, _ := ioutil.ReadDir(tmpdir)
			Expect(len(before)).To(BeNumerically("<=", len(after)))
		})
	})
})

type fakeCollector struct {
	started bool
	stopped bool
	stats   dea.Stats
}

func (f *fakeCollector) Start() bool {
	f.started = true
	return true
}
func (f *fakeCollector) Stop() {
	f.stopped = true
}
func (f *fakeCollector) GetStats() dea.Stats {
	return f.stats
}
func (f *fakeCollector) Retrieve_stats(now time.Time) {
}

type fakePromises struct {
	realPromises InstancePromises

	startInvoke            bool
	extractDropletInvoke   bool
	setupEnvironmentInvoke bool
	execHookScriptInvoke   bool
	stateInvoke            bool
	dropletInvoke          bool
	linkInvoke             bool
	linkPanic              bool
	crashHandlerInvoke     bool
	copyOutInvoke          bool

	healthcheckResponse bool
	healthcheckError    error

	stateFrom []dea.State
	stateTo   dea.State

	linkInvoked         bool
	crashHandlerInvoked bool
	copyOutInvoked      bool
	destroyInvoked      bool
}

func (f *fakePromises) Promise_start() error {
	if f.startInvoke {
		return f.realPromises.Promise_start()
	}
	return nil
}
func (f *fakePromises) Promise_copy_out() error {
	f.copyOutInvoked = true
	if f.copyOutInvoke {
		return f.realPromises.Promise_copy_out()
	}
	return nil
}
func (f *fakePromises) Promise_crash_handler() error {
	f.crashHandlerInvoked = true
	if f.crashHandlerInvoke {
		return f.realPromises.Promise_crash_handler()
	}
	return nil
}
func (f *fakePromises) Promise_container() error {
	return f.realPromises.Promise_container()
}
func (f *fakePromises) Promise_droplet() error {
	if f.dropletInvoke {
		return f.realPromises.Promise_droplet()
	}

	return nil
}
func (f *fakePromises) Promise_exec_hook_script(key string) error {
	if f.execHookScriptInvoke {
		return f.realPromises.Promise_exec_hook_script(key)
	}

	return nil
}
func (f *fakePromises) Promise_state(from []dea.State, to dea.State) error {
	f.stateFrom = from
	f.stateTo = to

	if f.stateInvoke {
		return f.realPromises.Promise_state(from, to)
	}
	return nil
}
func (f *fakePromises) Promise_extract_droplet() error {
	if f.extractDropletInvoke {
		return f.realPromises.Promise_extract_droplet()
	}
	return nil
}
func (f *fakePromises) Promise_setup_environment() error {
	if f.setupEnvironmentInvoke {
		return f.realPromises.Promise_setup_environment()
	}
	return nil
}
func (f *fakePromises) Link(callback func(error) error) {
	if f.linkInvoke {
		f.realPromises.Link(callback)
	}
}

func (f *fakePromises) Promise_link() (*warden.LinkResponse, error) {
	f.linkInvoked = true
	if f.linkPanic {
		panic("Link Panic")
	}

	if f.linkInvoke {
		return f.realPromises.Promise_link()
	}

	return nil, nil
}
func (f *fakePromises) Promise_read_instance_manifest(container_path string) (map[string]interface{}, error) {
	return nil, nil
}
func (f *fakePromises) Promise_health_check() (bool, error) {
	return f.healthcheckResponse, f.healthcheckError
}

func (f *fakePromises) Promise_stop() error {
	return nil
}
func (f *fakePromises) Promise_destroy() error {
	f.destroyInvoked = true
	return nil
}
