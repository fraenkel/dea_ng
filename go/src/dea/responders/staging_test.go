package responders

import (
	cfg "dea/config"
	"dea/droplet"
	"dea/loggregator"
	"dea/staging"
	temitter "dea/testhelpers/emitter"
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
	var config cfg.Config
	var subject *Staging
	var stagingRegistry *staging.StagingTaskRegistry
	var dropletRegistry *droplet.DropletRegistry
	var nats *fakeyagnats.FakeYagnats
	var staging_task *tstaging.MockStagingTask
	var appManager mockAppManager

	panicNewStagingTask := func(*cfg.Config, staging.StagingMessage, []staging.StagingBuildpack, *droplet.DropletRegistry, *steno.Logger) staging.StagingTask {
		panic("nasty error")
	}

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "staging")
		config = cfg.Config{BaseDir: tmpdir, WardenSocket: "/tmp/bogus"}
		dropletRegistry = droplet.NewDropletRegistry(tmpdir)
		stagingRegistry = staging.NewStagingTaskRegistry(staging.NewStagingTask)
		nats = fakeyagnats.New()
		staging_task = &tstaging.MockStagingTask{
			StagingMsg: staging.NewStagingMessage(map[string]interface{}{
				"app_id":  "some_app_id",
				"task_id": "task-id"}),
			Droplet_Sha1: "some-droplet-sha",
		}
		appManager = mockAppManager{}
	})

	JustBeforeEach(func() {
		subject = NewStaging(&appManager, nats, dea_id, stagingRegistry, &config, dropletRegistry, nil)
	})

	AfterEach(func() {
		subject.Stop()
		os.RemoveAll(config.BaseDir)
	})

	Describe("start", func() {
		Context("when config does not allow staging operations", func() {
			BeforeEach(func() {
				config.Staging.Enabled = false
			})

			It("does not subscribe to anything", func() {
				subject.Start()
				Expect(len(nats.Subscriptions)).To(Equal(0))
			})
		})

		Context("when the config allows staging operation", func() {
			BeforeEach(func() {
				config.Staging.Enabled = true
			})

			It("subscribes to 'staging' message", func() {
				subject.Start()
				Expect(len(nats.Subscriptions["staging"])).To(Equal(1))
			})

			It("subscribes to 'staging.<dea-id>.start' message", func() {
				subject.Start()
				Expect(len(nats.Subscriptions["staging."+dea_id+".start"])).To(Equal(1))
			})

			It("subscribes to 'staging.stop' message", func() {
				subject.Start()
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
				subject.Start()
			})

			It("unsubscribes from 'staging' message", func() {
				subs := nats.Subscriptions["staging"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				subject.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

			It("unsubscribes from 'staging.<dea_id>.start' message", func() {
				subs := nats.Subscriptions["staging."+dea_id+".start"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				subject.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

			It("unsubscribes from 'staging.stop' message", func() {
				subs := nats.Subscriptions["staging.stop"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]
				subject.Stop()
				Expect(len(nats.Unsubscriptions)).ToNot(Equal(0))
				Expect(nats.Unsubscriptions).To(ContainElement(sub.ID))
			})

		})
		Context("when subscription was not made", func() {
			It("does not unsubscribe", func() {
				subject.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(0))
			})
		})

	})

	Describe("handle", func() {
		Describe("staging async", func() {
			var message yagnats.Message

			newMockStagingTask := func(config *cfg.Config, staging_message staging.StagingMessage,
				buildpacksInUse []staging.StagingBuildpack, dropletRegistry *droplet.DropletRegistry, logger *steno.Logger) staging.StagingTask {
				return staging_task
			}

			BeforeEach(func() {
				stagingRegistry = staging.NewStagingTaskRegistry(newMockStagingTask)
				message = yagnats.Message{
					Subject: "subject",
					ReplyTo: "respond-to",
					Payload: []byte(`{"app_id":"app-id", "task_id":"task-id"}`),
				}
			})

			It("adds staging task to registry", func() {
				subject.handle(&message)
				Expect(stagingRegistry.Task("task-id")).NotTo(BeNil())
			})

			It("starts staging task with registered callbacks", func() {
				subject.handle(&message)
				Expect(staging_task.SetupCallback).ToNot(BeNil())
				Expect(staging_task.CompleteCallback).ToNot(BeNil())
				Expect(staging_task.UploadCallback).ToNot(BeNil())
				Expect(staging_task.StopCallback).ToNot(BeNil())
				Expect(staging_task.Started).To(BeTrue())
			})

			It("saves snapshot", func() {
				subject.handle(&message)
				Expect(appManager.savedSnapshot).To(BeTrue())
			})

			Describe("after staging container setup", func() {
				BeforeEach(func() {
					staging_task.Streaming_log_url = "stream-log-url"
				})

				Context("when staging succeeds setting up staging container", func() {
					It("responds with successful message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                "task-id",
							"task_streaming_log_url": "stream-log-url",
							"detected_buildpack":     nil,
							"error":                  nil,
							"droplet_sha1":           nil,
						}

						subject.handle(&message)
						Expect(staging_task.SetupCallback).NotTo(BeNil())
						staging_task.SetupCallback(nil)
						actualRsp := make(map[string]interface{})
						json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
						Expect(actualRsp).To(Equal(expectedRsp))
					})
				})

				Context("when staging fails to set up staging container", func() {
					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                "task-id",
							"task_streaming_log_url": "stream-log-url",
							"detected_buildpack":     nil,
							"error":                  "error-description",
							"droplet_sha1":           nil,
						}

						subject.handle(&message)
						Expect(staging_task.SetupCallback).NotTo(BeNil())
						staging_task.SetupCallback(errors.New("error-description"))
						actualRsp := make(map[string]interface{})
						json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
						Expect(actualRsp).To(Equal(expectedRsp))
					})
				})

			})

			Describe("after staging completion", func() {
				Context("when successfully", func() {
					app_id := "my_app_id"
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
						subject.handle(&message)
						Expect(staging_task.CompleteCallback).NotTo(BeNil())
						staging_task.CompleteCallback(nil)
						Expect(appManager.startedApp).To(Equal(start_message))
					})

					It("logs to the loggregator", func() {
						emitter := temitter.MockEmitter{}
						loggregator.SetEmitter(&emitter)

						subject.handle(&message)
						Expect(emitter.Messages[app_id][0]).To(Equal("Got staging request for app with id " + app_id))
					})
				})
				Context("when failed", func() {
					It("does not start an instance", func() {
						subject.handle(&message)
						Expect(staging_task.CompleteCallback).NotTo(BeNil())
						staging_task.CompleteCallback(errors.New("error-description"))
						Expect(len(appManager.startedApp)).To(Equal(0))
					})
				})
			})

			Describe("after staging upload", func() {
				Context("when successfully", func() {
					It("responds successful message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                "task-id",
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  nil,
							"droplet_sha1":           "some-droplet-sha",
						}

						subject.handle(&message)
						Expect(staging_task.UploadCallback).NotTo(BeNil())
						staging_task.UploadCallback(nil)
						actualRsp := make(map[string]interface{})
						json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
						Expect(actualRsp).To(Equal(expectedRsp))
					})

					It("unregisters task from registry", func() {
						subject.handle(&message)
						staging_task.UploadCallback(nil)
						Expect(stagingRegistry.Task("task-id")).To(BeNil())
					})

					It("saves snapshot", func() {
						subject.handle(&message)
						staging_task.UploadCallback(nil)
						Expect(appManager.savedSnapshot).To(BeTrue())
					})
				})

				Context("when failed", func() {
					err := errors.New("error-description")

					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                "task-id",
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  "error-description",
							"droplet_sha1":           "some-droplet-sha",
						}

						subject.handle(&message)
						Expect(staging_task.UploadCallback).NotTo(BeNil())
						staging_task.UploadCallback(err)
						actualRsp := make(map[string]interface{})
						json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
						Expect(actualRsp).To(Equal(expectedRsp))
					})

					It("unregisters task from registry", func() {
						subject.handle(&message)
						staging_task.UploadCallback(err)
						Expect(stagingRegistry.Task("task-id")).To(BeNil())
					})

					It("saves snapshot", func() {
						subject.handle(&message)
						staging_task.UploadCallback(err)
						Expect(appManager.savedSnapshot).To(BeTrue())
					})
				})

				Context("when stopped", func() {
					err := errors.New("Error staging: task stopped")

					It("responds with error message", func() {
						expectedRsp := map[string]interface{}{
							"task_id":                "task-id",
							"task_streaming_log_url": nil,
							"detected_buildpack":     nil,
							"error":                  "Error staging: task stopped",
							"droplet_sha1":           nil,
						}

						subject.handle(&message)
						Expect(staging_task.StopCallback).NotTo(BeNil())
						staging_task.StopCallback(err)
						actualRsp := make(map[string]interface{})
						json.Unmarshal(nats.PublishedMessages["respond-to"][0].Payload, &actualRsp)
						Expect(actualRsp).To(Equal(expectedRsp))
					})

					It("unregisters task from registry", func() {
						subject.handle(&message)
						staging_task.StopCallback(err)
						Expect(stagingRegistry.Task("task-id")).To(BeNil())
					})

					It("saves snapshot", func() {
						subject.handle(&message)
						staging_task.StopCallback(err)
						Expect(appManager.savedSnapshot).To(BeTrue())
					})
				})
			})

			Describe("when an error occurs", func() {
				BeforeEach(func() {
					stagingRegistry = staging.NewStagingTaskRegistry(panicNewStagingTask)
				})

				It("catches the error since this is the top level", func() {
					Expect(func() { subject.handle(&message) }).NotTo(Panic())
				})
			})
		})
	})

	Describe("handle", func() {
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
			subject.handleStop(&message)
			Expect(staging_task.Stopped).To(BeTrue())
		})

		Describe("when an error occurs", func() {
			BeforeEach(func() {
				stagingRegistry = staging.NewStagingTaskRegistry(panicNewStagingTask)
			})

			It("catches the error since this is the top level", func() {
				Expect(func() { subject.handleStop(&message) }).NotTo(Panic())
			})
		})
	})

})

type mockAppManager struct {
	startedApp    map[string]interface{}
	savedSnapshot bool
}

func (am *mockAppManager) StartApp(msg map[string]interface{}) {
	am.startedApp = msg
}
func (am *mockAppManager) SaveSnapshot() {
	am.savedSnapshot = true
}
