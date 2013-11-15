package responders

import (
	cfg "dea/config"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/staging"
	"dea/starting"
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
	var subject *DeaLocator
	var resourceManager resmgr.ResourceManager
	var instanceRegistry *starting.InstanceRegistry
	var stagingRegistry *staging.StagingTaskRegistry
	var nats *fakeyagnats.FakeYagnats

	BeforeEach(func() {
		tmpdir, _ := ioutil.TempDir("", "dea_locator")
		config = cfg.Config{BaseDir: tmpdir}
		instanceRegistry = starting.NewInstanceRegistry(&config)
		stagingRegistry = staging.NewStagingTaskRegistry(staging.NewStagingTask)
		resourceManager = resmgr.NewResourceManager(instanceRegistry, stagingRegistry, &config.Resources)
		nats = fakeyagnats.New()
	})

	JustBeforeEach(func() {
		subject = NewDeaLocator(nats, dea_id, resourceManager, &config)
	})

	AfterEach(func() {
		subject.Stop()
		os.RemoveAll(config.BaseDir)
	})

	Describe("start", func() {
		Describe("subscription for 'dea.locate'", func() {
			It("subscribes to 'dea.locate' message", func() {
				subject.Start()
				Expect(len(nats.Subscriptions["dea.locate"])).To(Equal(1))
			})
		})

		It("it publishes dea.advertise on dea.locate message", func() {
			subject.Start()

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
				subject.Start()
				time.Sleep(20 * time.Millisecond)

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
				Expect(subject.advertiseIntervals).To(Equal(default_advertise_interval))
			})
		})
	})

	Describe("stop", func() {
		BeforeEach(func() {
			config.Intervals.Advertise = 10 * time.Millisecond
		})

		Context("when subscription was made", func() {
			It("unsubscribes from 'dea.locate' message", func() {
				subject.Start()
				subs := nats.Subscriptions["dea.locate"]
				Expect(len(subs)).To(Equal(1))
				sub := subs[0]

				subject.Stop()
				Expect(len(nats.Unsubscriptions)).To(Equal(1))
				Expect(nats.Unsubscriptions[0]).To(Equal(sub.ID))
			})

			It("stops sending 'dea.advertise' periodically", func() {
				subject.Start()
				subject.Stop()
				before := len(nats.PublishedMessages["dea.advertise"])
				time.Sleep(20 * time.Millisecond)
				after := len(nats.PublishedMessages["dea.advertise"])
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
		BeforeEach(func() {
			//      resource_manager.stub(:app_id_to_count => {
			//          "app_id_1" => 1,
			//          "app_id_2" => 3
			//      })
		})

		Context("when config specifies stacks", func() {
			availableMemory := 12345.0
			availableDisk := 45678.0
			appCounts := map[string]int{
				"app_id_1": 1,
				"app_id_2": 3,
			}
			stacks := []string{"stack-1", "stack-2"}

			BeforeEach(func() {
				config.Stacks = stacks
				resourceManager = mockResourceManager{
					memory:    availableMemory,
					disk:      availableDisk,
					appCounts: appCounts,
				}
			})

			It("publishes 'dea.advertise' message", func() {
				subject.Start()
				subject.Advertise()
				subject.Stop()
				advertised := nats.PublishedMessages["dea.advertise"]
				Expect(len(advertised)).To(Equal(1))
				advertiseMsg := protocol.AdvertiseMessage{}
				err := json.Unmarshal(advertised[0].Payload, &advertiseMsg)
				Expect(err).To(BeNil())
				expectedMsg := protocol.NewAdvertiseMessage(dea_id, stacks, availableMemory, availableDisk, appCounts)
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

type mockResourceManager struct {
	memory    float64
	disk      float64
	appCounts map[string]int
}

func (rm mockResourceManager) MemoryCapacity() float64 {
	return 1.0
}
func (rm mockResourceManager) DiskCapacity() float64 {
	return 1.0
}
func (rm mockResourceManager) AppIdToCount() map[string]int {
	return rm.appCounts
}
func (rm mockResourceManager) RemainingMemory() float64 {
	return rm.memory
}
func (rm mockResourceManager) ReservedMemory() float64 {
	return 1.0
}
func (rm mockResourceManager) UsedMemory() float64 {
	return 1.0
}
func (rm mockResourceManager) CanReserve(memory, disk float64) bool {
	return true
}
func (rm mockResourceManager) RemainingDisk() float64 {
	return rm.disk
}
func (rm mockResourceManager) NumberReservable(memory, disk uint64) uint {
	return 1
}
func (rm mockResourceManager) AvailableMemoryRatio() float64 {
	return 1.0
}
func (rm mockResourceManager) AvailableDiskRatio() float64 {
	return 1.0
}
