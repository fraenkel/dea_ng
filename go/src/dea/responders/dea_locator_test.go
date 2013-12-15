package responders

import (
	"dea"
	cfg "dea/config"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/staging"
	"dea/starting"
	testrm "dea/testhelpers/resource_manager"
	"encoding/json"
	"errors"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"time"
)

var _ = Describe("DeaLocator", func() {
	dea_id := "unique-dea-id"
	var config cfg.Config
	var deaLocator *DeaLocator
	var resourceManager dea.ResourceManager
	var instanceRegistry dea.InstanceRegistry
	var stagingRegistry dea.StagingTaskRegistry
	var nats *fakeyagnats.FakeYagnats

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "dea_locator")
		config = cfg.Config{BaseDir: tmpdir}
		instanceRegistry = starting.NewInstanceRegistry(&config)
		stagingRegistry = staging.NewStagingTaskRegistry(&config, nil, staging.NewStagingTask)
		resourceManager = resmgr.NewResourceManager(instanceRegistry, stagingRegistry, &config.Resources)
		nats = fakeyagnats.New()
	})

	JustBeforeEach(func() {
		deaLocator = NewDeaLocator(nats, dea_id, resourceManager, &config)
	})

	AfterEach(func() {
		deaLocator.Stop()
		os.RemoveAll(config.BaseDir)
	})

	Describe("start", func() {
		Describe("subscription for 'dea.locate'", func() {
			It("subscribes to 'dea.locate' message", func() {
				deaLocator.Start()
				Expect(len(nats.Subscriptions["dea.locate"])).To(Equal(1))
			})
		})

		It("it publishes dea.advertise on dea.locate message", func() {
			deaLocator.Start()

			publish(nats, "dea.locate", nil, "")

			advertised := nats.PublishedMessages["dea.advertise"]
			Expect(len(advertised)).To(Equal(1))
		})

	})

	Describe("periodic 'dea.advertise'", func() {

		Context("when intervals.advertise config is set", func() {
			BeforeEach(func() {
				config.Intervals.Advertise = 10 * time.Millisecond
			})
			It("starts sending every 10ms", func() {
				deaLocator.Start()
				time.Sleep(30 * time.Millisecond)

				advertised := nats.PublishedMessages["dea.advertise"]
				Expect(len(advertised)).To(BeNumerically(">", 1))
				Expect(len(advertised)).To(BeNumerically("<", 4))
			})
		})

		Context("when intervals.advertise config is not set", func() {
			BeforeEach(func() {
				config.Intervals.Advertise = 0
			})

			It("starts sending every 5 seconds", func() {
				Expect(deaLocator.advertiseIntervals).To(Equal(default_advertise_interval))
			})
		})
	})

	Describe("stop", func() {
		BeforeEach(func() {
			config.Intervals.Advertise = 10 * time.Millisecond
		})

		Context("when subscription was made", func() {
			It("unsubscribes from 'dea.locate' message", func() {
				deaLocator.Start()
				subs := nats.Subscriptions["dea.locate"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]

				deaLocator.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(1))
				Expect(nats.Unsubscriptions[0]).To(Equal(sub.ID))
			})

			It("stops sending 'dea.advertise' periodically", func() {
				deaLocator.Start()
				deaLocator.Stop()
				before := len(nats.PublishedMessages["dea.advertise"])
				time.Sleep(20 * time.Millisecond)
				after := len(nats.PublishedMessages["dea.advertise"])
				Expect(after).To(Equal(before))
			})
		})

		Context("when subscription was not made", func() {
			It("does not unsubscribe", func() {
				deaLocator.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(0))
			})

		})
	})

	Describe("advertise", func() {
		availableMemory := 12345.0
		availableDisk := 45678.0
		appCounts := map[string]int{
			"app_id_1": 1,
			"app_id_2": 3,
		}
		stacks := []string{"stack-1", "stack-2"}

		BeforeEach(func() {
			config.PlacementProperties = cfg.PlacementConfig{Zone: "zone1"}
			config.Stacks = stacks

			resourceManager = testrm.FakeResourceManager{
				Memory:    availableMemory,
				Disk:      availableDisk,
				AppCounts: appCounts,
			}
		})

		advertise := func() protocol.AdvertiseMessage {
			deaLocator.Advertise()
			advertised := nats.PublishedMessages["dea.advertise"]
			Expect(len(advertised)).To(Equal(1))
			advertiseMsg := protocol.AdvertiseMessage{}
			err := json.Unmarshal(advertised[0].Payload, &advertiseMsg)
			Expect(err).To(BeNil())
			return advertiseMsg
		}

		It("publishes 'dea.advertise' message", func() {
			msg := advertise()
			Expect(msg).To(Equal(protocol.AdvertiseMessage{
				ID:                  dea_id,
				Stacks:              stacks,
				AvailableMemory:     availableMemory,
				AvailableDisk:       availableDisk,
				AppCounts:           appCounts,
				PlacementProperties: protocol.PlacementProperties{Zone: "zone1"},
			}))
		})

		Context("when a failure happens", func() {
			It("should catch the error since this is the top level", func() {
				defer func() {
					r := recover()
					Expect(r).To(BeNil())
				}()
				nats.PublishError = errors.New("Something terrible happened")
				advertise()
			})
		})
	})
})

func publish(nats *fakeyagnats.FakeYagnats, subject string, payload []byte, replyTo string) {
	msg := &yagnats.Message{
		Subject: subject,
		Payload: payload,
		ReplyTo: replyTo,
	}

	for _, sub := range nats.Subscriptions[subject] {
		sub.Callback(msg)
	}
}
