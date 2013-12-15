package lifecycle

import (
	"dea"
	cfg "dea/config"
	"dea/protocol"
	"dea/starting"
	thelpers "dea/testhelpers"
	tlogger "dea/testhelpers/logger"
	tresponder "dea/testhelpers/responders"
	tstarting "dea/testhelpers/starting"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
	"time"
)

var _ = Describe("EvacuationHandler", func() {
	var config cfg.Config
	var evacHandler *evacuationHandler
	var fakenats *fakeyagnats.FakeYagnats
	var fakeresponder *tresponder.FakeResponder
	var fakel tlogger.FakeL
	var iRegistry *tstarting.FakeInstanceRegistry

	var instances map[dea.State]dea.Instance

	goodbyeMessage := protocol.NewGoodbyeMessage("bogus_uuid", "127.0.0.1", nil)

	createInstance := func(state dea.State, m map[string]interface{}) dea.Instance {
		attributes := thelpers.Valid_instance_attributes(false)
		for k, v := range m {
			attributes[k] = v
		}
		instance := starting.NewInstance(attributes, &config, nil, "127.0.0.1")
		instance.SetState(state)
		return instance
	}

	BeforeEach(func() {
		config, _ = cfg.NewConfig(nil)

		fakenats = fakeyagnats.New()
		fakeresponder = &tresponder.FakeResponder{}

		fakel = tlogger.FakeL{}
		iRegistry = tstarting.NewFakeInstanceRegistry()

		instances = make(map[dea.State]dea.Instance)
		for _, s := range starting.STATES {
			i := createInstance(s, nil)
			instances[s] = i
		}
	})

	JustBeforeEach(func() {
		logger := utils.Logger("evac", nil)
		logger.L = &fakel

		evacHandler = NewEvacuationHandler(15*time.Second, []dea.Responder{fakeresponder}, iRegistry, fakenats, logger)
	})

	Context("before the evacuation handler is called", func() {
		It("is not evacuating", func() {
			Expect(evacHandler.isEvacuating()).To(BeFalse())
		})
	})

	Context("when the evacuation handler is called for the first time", func() {
		JustBeforeEach(func() {
			evacHandler.Evacuate(goodbyeMessage)
		})

		It("is evacuating", func() {
			Expect(evacHandler.isEvacuating()).To(BeTrue())
		})

		It("sends the shutdown message", func() {
			Expect(fakenats.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
			Expect(string(fakenats.PublishedMessages["dea.shutdown"][0].Payload)).To(ContainSubstring("bogus_uuid"))
		})

		It("stops advertising", func() {
			Expect(fakeresponder.Stopped).To(BeTrue())
		})
	})

	Context("with a mixture of instances in various states", func() {
		BeforeEach(func() {
			for _, i := range instances {
				iRegistry.Register(i)
			}
		})

		It("sends the exit message for all born/starting/running/resuming instances (will be removed when deterministic evacuation is complete)", func() {
			evacHandler.Evacuate(goodbyeMessage)

			messages := fakenats.PublishedMessages["droplet.exited"]
			Expect(messages).To(HaveLen(4))

			actualIds := make(map[string]struct{})
			for _, m := range messages {
				exitm := protocol.ExitMessage{}
				json.Unmarshal(m.Payload, &exitm)
				actualIds[exitm.Id] = struct{}{}
				Expect(exitm.Reason).To(Equal(EXIT_REASON_EVACUATION))
			}

			expectedIds := make(map[string]struct{})
			expectedIds[instances[dea.STATE_BORN].Id()] = struct{}{}
			expectedIds[instances[dea.STATE_STARTING].Id()] = struct{}{}
			expectedIds[instances[dea.STATE_RUNNING].Id()] = struct{}{}
			expectedIds[instances[dea.STATE_RESUMING].Id()] = struct{}{}
			Expect(actualIds).To(Equal(expectedIds))
		})

		It("transitions born/running/starting/resuming instances to evacuating", func() {
			evacHandler.Evacuate(goodbyeMessage)

			Expect(instances[dea.STATE_BORN].State()).To(Equal(dea.STATE_EVACUATING))
			Expect(instances[dea.STATE_STARTING].State()).To(Equal(dea.STATE_EVACUATING))
			Expect(instances[dea.STATE_RUNNING].State()).To(Equal(dea.STATE_EVACUATING))
			Expect(instances[dea.STATE_RESUMING].State()).To(Equal(dea.STATE_EVACUATING))
		})

		It("returns false", func() {
			Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeFalse())
		})
	})

	Context("with all instances either stopping/stopped/crashed/deleted", func() {
		BeforeEach(func() {
			iRegistry.Register(instances[dea.STATE_STOPPING])
			iRegistry.Register(instances[dea.STATE_STOPPED])
			iRegistry.Register(instances[dea.STATE_CRASHED])
			iRegistry.Register(instances[dea.STATE_DELETED])
		})

		It("does not transition instances", func() {
			evacHandler.Evacuate(goodbyeMessage)

			Expect(instances[dea.STATE_STOPPING].State()).To(Equal(dea.STATE_STOPPING))
			Expect(instances[dea.STATE_STOPPED].State()).To(Equal(dea.STATE_STOPPED))
			Expect(instances[dea.STATE_CRASHED].State()).To(Equal(dea.STATE_CRASHED))
			Expect(instances[dea.STATE_DELETED].State()).To(Equal(dea.STATE_DELETED))
		})

		It("returns true", func() {
			Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeTrue())
		})

	})

	Context("when the dea has no instances", func() {
		It("returns true", func() {
			Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeTrue())
		})
	})

	Context("and its called again", func() {
		var now time.Time
		JustBeforeEach(func() {
			now = time.Now()
			evacHandler.Evacuate(goodbyeMessage)
			delete(fakenats.PublishedMessages, "dea.shutdown")
		})

		It("does not send the shutdown message", func() {
			evacHandler.Evacuate(goodbyeMessage)
			Expect(fakenats.PublishedMessages["dea.shutdown"]).To(BeNil())
		})

		Context("and the registry still has evacuating instances", func() {
			BeforeEach(func() {
				iRegistry.Register(instances[dea.STATE_EVACUATING])
			})

			It("returns false (the dea should not be stopped yet)", func() {
				evacHandler.endTime = now.Add(10 * time.Minute)
				Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeFalse())
			})
		})

		It("logs nothing", func() {
			evacHandler.Evacuate(goodbyeMessage)
			Expect(fakel.Logs["error"]).To(BeNil())
		})

		It("should not send exit messages", func() {
			evacHandler.Evacuate(goodbyeMessage)
			Expect(fakenats.PublishedMessages["droplet.exited"]).To(BeNil())
		})

		Context("and the time elapsed since the first call is greater than the configured time", func() {
			It("returns true", func() {
				evacHandler.endTime = now.Add(15 * time.Minute)
				Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeTrue())
			})
		})

		Context("and the registry (somehow) has born/starting/resuming/running instances", func() {
			JustBeforeEach(func() {
				iRegistry.Register(instances[dea.STATE_BORN])
				iRegistry.Register(instances[dea.STATE_STARTING])
				iRegistry.Register(instances[dea.STATE_RESUMING])
				iRegistry.Register(instances[dea.STATE_RUNNING])
			})

			It("should send exit messages", func() {
				evacHandler.Evacuate(goodbyeMessage)
				messages := fakenats.PublishedMessages["droplet.exited"]
				Expect(messages).To(HaveLen(4))
			})

			It("transitions any lagging instances to evacuating (we presume this should never happen)", func() {
				evacHandler.Evacuate(goodbyeMessage)

				Expect(instances[dea.STATE_BORN].State()).To(Equal(dea.STATE_EVACUATING))
				Expect(instances[dea.STATE_STARTING].State()).To(Equal(dea.STATE_EVACUATING))
				Expect(instances[dea.STATE_RUNNING].State()).To(Equal(dea.STATE_EVACUATING))
				Expect(instances[dea.STATE_RESUMING].State()).To(Equal(dea.STATE_EVACUATING))
			})

			It("logs this faulty state", func() {
				evacHandler.Evacuate(goodbyeMessage)
				cnt := 0
				for _, m := range fakel.Logs["error"] {
					if strings.Contains(m, "Found an unexpected") {
						cnt++
					}
				}
				Expect(cnt).To(Equal(4))
			})

			It("returns false (the dea should not be stopped yet)", func() {
				evacHandler.endTime = now.Add(10 * time.Minute)
				Expect(evacHandler.Evacuate(goodbyeMessage)).To(BeFalse())
			})
		})
	})
})
