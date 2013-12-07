package responders

import (
	cfg "dea/config"
	"dea/droplet"
	"dea/loggregator"
	stg "dea/staging"
	tboot "dea/testhelpers/boot"
	temitter "dea/testhelpers/emitter"
	trm "dea/testhelpers/resource_manager"
	tstaging "dea/testhelpers/staging"
	"encoding/json"
	"errors"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("Staging", func() {
	dea_id := "unique-dea-id"
	app_id := "my_app_id"
	task_id := "task-id"
	var config cfg.Config
	var boot tboot.FakeBootstrap
	var staging_task *tstaging.FakeStagingTask
	var nats *fakeyagnats.FakeYagnats
	var stagingRegistry *stg.StagingTaskRegistry
	var snapshot tboot.FakeSnapshot
	var fakeim tboot.FakeInstanceManager
	var staging *Staging
	var resmgr trm.FakeResourceManager

	panicNewStagingTask := func(*cfg.Config, stg.StagingMessage, []stg.StagingBuildpack, droplet.DropletRegistry, *steno.Logger) stg.StagingTask {
		panic("nasty error")
	}

	newStagingTask := func() *tstaging.FakeStagingTask {
		task := &tstaging.FakeStagingTask{
			StagingMsg: stg.NewStagingMessage(map[string]interface{}{
				"app_id":  "some_app_id",
				"task_id": task_id,
			}),
			Droplet_Sha1: "some-droplet-sha",
			StagingCfg: cfg.StagingConfig{
				MemoryLimitMB: 1,
				DiskLimitMB:   2,
			},
		}
		return task
	}

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "staging")
		config, _ = cfg.NewConfig(nil)
		config.BaseDir = tmpdir
		config.WardenSocket = "/tmp/bogus"

		nats = fakeyagnats.New()
		stagingRegistry = stg.NewStagingTaskRegistry(stg.NewStagingTask)
		snapshot = tboot.FakeSnapshot{}
		fakeim = tboot.FakeInstanceManager{}
		resmgr = trm.FakeResourceManager{}

		staging_task = newStagingTask()
	})

	JustBeforeEach(func() {
		boot = tboot.FakeBootstrap{
			Cfg:             config,
			IManager:        &fakeim,
			DropRegistry:    droplet.NewDropletRegistry(config.BaseDir),
			StagingRegistry: stagingRegistry,
			NatsClient:      nats,
			Snap:            &snapshot,
			ResMgr:          &resmgr,
		}

		staging = NewStaging(&boot, dea_id, nil)
	})

	AfterEach(func() {
		staging.Stop()
		os.RemoveAll(config.BaseDir)
	})

	Describe("start", func() {
		Context("when config does not allow staging operations", func() {
			BeforeEach(func() {
				config.Staging.Enabled = false
			})

			It("does not subscribe to anything", func() {
				staging.Start()
				Expect(len(nats.Subscriptions)).To(Equal(0))
			})
		})

		Context("when the config allows staging operation", func() {
			BeforeEach(func() {
				config.Staging.Enabled = true
			})

			It("subscribes to 'staging' message", func() {
				staging.Start()
				Expect(len(nats.Subscriptions["staging"])).To(Equal(1))
			})

			It("subscribes to 'staging.<dea-id>.start' message", func() {
				staging.Start()
				Expect(len(nats.Subscriptions["staging."+dea_id+".start"])).To(Equal(1))
			})

			It("subscribes to 'staging.stop' message", func() {
				staging.Start()
				Expect(len(nats.Subscriptions["staging.stop"])).To(Equal(1))
			})

		})
	})

	Describe("stop", func() {
		BeforeEach(func() {
			config.Staging.Enabled = true
		})

		Context("when subscription was made", func() {
			JustBeforeEach(func() {
				staging.Start()
			})

			It("unsubscribes from 'staging' message", func() {
				subs := nats.Subscriptions["staging"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				staging.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

			It("unsubscribes from 'staging.<dea_id>.start' message", func() {
				subs := nats.Subscriptions["staging."+dea_id+".start"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				staging.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

			It("unsubscribes from 'staging.stop' message", func() {
				subs := nats.Subscriptions["staging.stop"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				staging.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

		})
		Context("when subscription was not made", func() {
			It("does not unsubscribe", func() {
				staging.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(0))
			})
		})

	})

	Describe("handle", func() {
		var message yagnats.Message

		it_registers_task := func() {
			It("adds staging task to registry", func() {
				staging.handle(&message)
				Expect(stagingRegistry.Task(task_id)).NotTo(BeNil())
			})
		}

		it_unregisters_task := func() {
			It("unregisters staging task from registry", func() {
				staging.handle(&message)
				Expect(stagingRegistry.Task(task_id)).To(BeNil())
			})
		}

		it_handles_and_responds := func(expectedRsp map[string]interface{}) {
			staging.handle(&message)
			actualRsp := make(map[string]interface{})
			json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
			Expect(actualRsp).To(Equal(expectedRsp))
		}

		BeforeEach(func() {
			message = yagnats.Message{
				Subject: "subject",
				ReplyTo: "respond-to",
				Payload: []byte(`{"app_id":"` + app_id + `", "task_id":"` + task_id + `"}`),
			}
		})

		It("logs to the loggregator", func() {
			emitter := temitter.FakeEmitter{}
			loggregator.SetEmitter(&emitter)

			staging.handle(&message)
			Expect(emitter.Messages[app_id][0]).To(Equal("Got staging request for app with id " + app_id))
		})

		Context("staging async", func() {
			newFakeStagingTask := func(config *cfg.Config, staging_message stg.StagingMessage,
				buildpacksInUse []stg.StagingBuildpack, dropletRegistry droplet.DropletRegistry, logger *steno.Logger) stg.StagingTask {
				return staging_task
			}

			BeforeEach(func() {
				stagingRegistry = stg.NewStagingTaskRegistry(newFakeStagingTask)
			})

			It("starts staging task with registered callbacks", func() {
				staging.handle(&message)
				Expect(staging_task.SetupCallback).ToNot(BeNil())
				Expect(staging_task.CompleteCallback).ToNot(BeNil())
				Expect(staging_task.StopCallback).ToNot(BeNil())
				Expect(staging_task.Started).To(BeTrue())
			})

			it_registers_task()

			It("saves snapshot", func() {
				staging.handle(&message)
				Expect(snapshot.SaveInvoked).To(BeTrue())
			})

			Describe("after staging container setup", func() {
				BeforeEach(func() {
					staging_task.Streaming_log_url = "stream-log-url"
				})

				Context("when staging succeeds setting up staging container", func() {
					BeforeEach(func() {
						staging_task.InvokeSetup = true
					})

					It("responds with successful message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": "stream-log-url",
							"detected_buildpack":     nil,
							"error":                  nil,
							"droplet_sha1":           nil,
						}

						it_handles_and_responds(expectedRsp)
					})
				})

				Context("when staging fails to set up staging container", func() {
					BeforeEach(func() {
						staging_task.InvokeSetup = true
						staging_task.SetupError = errors.New("error-description")
					})

					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": "stream-log-url",
							"detected_buildpack":     nil,
							"error":                  "error-description",
							"droplet_sha1":           nil,
						}

						it_handles_and_responds(expectedRsp)
					})
				})

			})

			Describe("after staging completed", func() {
				Context("when successful", func() {
					BeforeEach(func() {
						staging_task.InvokeComplete = true
					})

					It("responds with successful message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  nil,
							"droplet_sha1":           "some-droplet-sha",
						}

						it_handles_and_responds(expectedRsp)
					})

					it_unregisters_task()

					It("saves snapshot", func() {
						staging.handle(&message)
						Expect(snapshot.SaveInvoked).To(BeTrue())
					})

					Context("when there is a start message in staging message", func() {
						start_message := map[string]interface{}{
							"droplet": "dff77854-3767-41d9-ab16-c8a824beb77a",
							"sha1":    "some-droplet-sha",
						}

						BeforeEach(func() {
							msg := map[string]interface{}{
								"app_id":        app_id,
								"start_message": start_message,
							}
							msgBytes, _ := json.Marshal(msg)
							message = yagnats.Message{
								Subject: "subject",
								ReplyTo: "respond-to",
								Payload: msgBytes,
							}
						})

						It("handles instance start with updated droplet sha", func() {
							staging.handle(&message)
							Expect(staging_task.CompleteCallback).NotTo(BeNil())
							Expect(fakeim.StartAppInvoked).To(BeTrue())
							Expect(fakeim.StartAppAttributes).To(Equal(start_message))
						})

					})
				})

				Context("when failed", func() {
					BeforeEach(func() {
						staging_task.InvokeComplete = true
						staging_task.CompleteError = errors.New("error-description")
					})

					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  "error-description",
							"droplet_sha1":           "some-droplet-sha",
						}

						it_handles_and_responds(expectedRsp)
					})

					it_unregisters_task()

					It("unregisters task from registry", func() {
						staging.handle(&message)
						Expect(stagingRegistry.Task(task_id)).To(BeNil())
					})

					It("saves snapshot", func() {
						staging.handle(&message)
						Expect(snapshot.SaveInvoked).To(BeTrue())
					})

					It("does not start an instance", func() {
						staging.handle(&message)
						Expect(fakeim.StartAppInvoked).To(BeFalse())
					})
				})
			})

			Context("when stopped", func() {
				BeforeEach(func() {
					staging_task.InvokeStop = true
				})

				Context("when successful", func() {
					It("responds with successful message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  nil,
							"droplet_sha1":           nil,
						}

						it_handles_and_responds(expectedRsp)
					})
				})

				Context("when failed", func() {
					BeforeEach(func() {
						staging_task.InvokeStop = true
						staging_task.StopError = errors.New("Error staging: task stopped")
					})

					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                task_id,
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  "Error staging: task stopped",
							"droplet_sha1":           nil,
						}

						it_handles_and_responds(expectedRsp)
					})
				})

				it_unregisters_task()

				It("saves snapshot", func() {
					staging.handle(&message)
					Expect(snapshot.SaveInvoked).To(BeTrue())
				})
			})
		})

		Context("when an error occurs during staging", func() {
			BeforeEach(func() {
				stagingRegistry = stg.NewStagingTaskRegistry(panicNewStagingTask)
			})

			It("catches the error since this is the top level", func() {
				Expect(func() { staging.handle(&message) }).NotTo(Panic())
			})
		})

		Context("when not enough resources available", func() {
			newFakeStagingTask := func(config *cfg.Config, staging_message stg.StagingMessage,
				buildpacksInUse []stg.StagingBuildpack, dropletRegistry droplet.DropletRegistry, logger *steno.Logger) stg.StagingTask {
				return staging_task
			}

			BeforeEach(func() {
				resmgr.ConstrainedResource = "memory"
				stagingRegistry = stg.NewStagingTaskRegistry(newFakeStagingTask)
			})

			It("does not register staging task", func() {
				staging.handle(&message)
				Expect(stagingRegistry.Tasks()).To(HaveLen(0))
			})

			It("does not start staging task", func() {
				staging.handle(&message)
				Expect(staging_task.Started).Should(BeFalse())
			})

			It("responds with error message", func() {
				expectedRsp := map[string]interface{}{
					"task_id":                task_id,
					"task_streaming_log_url": nil,
					"detected_buildpack":     nil,
					"error":                  "Not enough memory resources available",
					"droplet_sha1":           nil,
				}

				it_handles_and_responds(expectedRsp)
			})
		})
	})

	Describe("handleStop", func() {
		var message yagnats.Message

		BeforeEach(func() {
			message = yagnats.Message{
				Subject: "subject",
				ReplyTo: "respond-to",
				Payload: []byte(`{"app_id":"some_app_id"}`),
			}

			stagingRegistry.Register(staging_task)
		})

		It("stops all staging tasks with the given id", func() {
			staging.handleStop(&message)
			Expect(staging_task.Stopped).To(BeTrue())
		})

		Describe("when an error occurs", func() {
			BeforeEach(func() {
				stagingRegistry = stg.NewStagingTaskRegistry(panicNewStagingTask)
			})

			It("catches the error since this is the top level", func() {
				Expect(func() { staging.handleStop(&message) }).NotTo(Panic())
			})
		})
	})

})
