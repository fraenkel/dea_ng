package responders

import (
	cfg "dea/config"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/staging"
	"dea/starting"
	testrm "dea/testhelpers/resource_manager"
	"encoding/json"
	"errors"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"time"
)

var _ = Describe("StagingLocator", func() {
	dea_id := "unique-dea-id"
	var nats *fakeyagnats.FakeYagnats
	var config cfg.Config
	var subject *StagingLocator
	var instanceRegistry *starting.InstanceRegistry
	var stagingRegistry *staging.StagingTaskRegistry

	var resourceManager resmgr.ResourceManager

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "staging_locator")
		config = cfg.Config{BaseDir: tmpdir}
		instanceRegistry = starting.NewInstanceRegistry(&config)
		stagingRegistry = staging.NewStagingTaskRegistry(staging.NewStagingTask)
		resourceManager = resmgr.NewResourceManager(instanceRegistry, stagingRegistry, &config.Resources)
		nats = fakeyagnats.New()
	})

	AfterEach(func() {
		subject.Stop()
		os.RemoveAll(config.BaseDir)
	})

	JustBeforeEach(func() {
		subject = NewStagingLocator(nats, dea_id, resourceManager, &config)
	})

	Describe("Start", func() {

		Describe("subscription for 'staging.locate'", func() {

			It("subscribes to 'staging.locate' message", func() {
				subject.Start()

				publish(nats, "staging.locate", nil, "")

				advertised := nats.PublishedMessages["staging.advertise"]
				Expect(len(advertised)).To(Equal(1))
			})

			It("subscribes to locate message", func() {
				subject.Start()
				Expect(len(nats.Subscriptions["staging.locate"])).To(Equal(1))
			})
		})

		Describe("periodic 'staging.advertise'", func() {
			Context("when intervals.advertise config is set", func() {
				BeforeEach(func() {
					config.Intervals.Advertise = 10 * time.Millisecond
				})

				It("starts sending every 10ms", func() {
					subject.Start()
					time.Sleep(30 * time.Millisecond)

					advertised := nats.PublishedMessages["staging.advertise"]
					Expect(len(advertised)).To(BeNumerically(">", 1))
					Expect(len(advertised)).To(BeNumerically("<", 4))
				})
			})

			Context("when intervals.advertise config is not set", func() {
				BeforeEach(func() {
					config.Intervals.Advertise = 0
				})

				It("starts sending every 5 seconds", func() {
					Expect(subject.advertiseIntervals).To(Equal(default_advertise_interval))
				})
			})
		})
	})

	Describe("stop", func() {
		BeforeEach(func() {
			config.Intervals.Advertise = 10 * time.Millisecond
		})

		Context("when subscription was made", func() {
			It("unsubscribes from 'staging.locate' message", func() {
				subject.Start()
				subs := nats.Subscriptions["staging.locate"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]

				subject.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(1))
				Expect(nats.Unsubscriptions[0]).To(Equal(sub.ID))
			})

			It("stops sending 'staging.advertise' periodically", func() {
				subject.Start()
				subject.Stop()
				before := len(nats.PublishedMessages["staging.advertise"])
				time.Sleep(20 * time.Millisecond)
				after := len(nats.PublishedMessages["staging.advertise"])
				Expect(after).To(Equal(before))
			})
		})

		Context("when subscription was not made", func() {
			It("does not unsubscribe", func() {
				subject.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(0))
			})

		})
	})

	Describe("advertise", func() {
		Context("when config specifies stacks", func() {
			availableMemory := 45678.0
			availableDisk := 45678.0
			appCounts := map[string]int{
				"app_id_1": 1,
				"app_id_2": 3,
			}
			stacks := []string{"lucid64"}

			BeforeEach(func() {
				config.Stacks = stacks
				resourceManager = testrm.MockResourceManager{
					Memory:    availableMemory,
					Disk:      availableDisk,
					AppCounts: appCounts,
				}
			})

			It("publishes 'staging.advertise' message", func() {
				subject.Start()
				subject.Advertise()
				subject.Stop()
				advertised := nats.PublishedMessages["staging.advertise"]
				Expect(len(advertised)).To(Equal(1))
				advertiseMsg := protocol.AdvertiseMessage{}
				err := json.Unmarshal(advertised[0].Payload, &advertiseMsg)
				Expect(err).To(BeNil())
				expectedMsg := protocol.NewAdvertiseMessage(dea_id, stacks, availableMemory, availableDisk, nil)
				Expect(advertiseMsg).To(Equal(*expectedMsg))
			})

		})

		Context("when a failure happens", func() {
			It("should catch the error since this is the top level", func() {
				defer func() {
					r := recover()
					Expect(r).To(BeNil())
				}()
				nats.PublishError = errors.New("Something terrible happened")
				subject.Start()
				subject.Advertise()
				subject.Stop()
			})
		})
	})
})
