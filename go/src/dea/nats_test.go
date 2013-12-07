package dea_test

import (
	. "dea"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nats", func() {
	var nats *Nats
	var fakeNats *fakeyagnats.FakeYagnats
	var natsHandler NatsHandler

	subjects := []string{"healthmanager.start", "router.start", "dea.status", "dea.UUID.start",
		"dea.stop", "dea.update", "dea.find.droplet"}

	BeforeEach(func() {
		nats, _ = NewNats([]string{"nats://user:password@myexample.com:4222"})
		fakeNats = fakeyagnats.New()
		nats.NatsClient = fakeNats

		natsHandler = &MockHandler{}
		nats.Start(natsHandler)

	})

	Describe("Start", func() {
		It("should connect", func() {
			Expect(fakeNats.ConnectedConnectionProvider).ShouldNot(BeNil())
		})

		Context("subscription setup", func() {
			It("should subscribe to ...", func() {
				Expect(len(fakeNats.Subscriptions)).To(Equal(len(subjects)))
				for _, s := range subjects {
					_, exists := fakeNats.Subscriptions[s]
					Expect(exists).To(BeTrue())
				}
			})
		})
	})

	Describe("Stop", func() {
		Context("subscription teardown", func() {

			It("should unsubscribe from everything when stop is called", func() {
				nats.Stop()
				Expect(len(fakeNats.Unsubscriptions)).To(Equal(len(subjects)))

				for _, s := range subjects {
					subs, _ := fakeNats.Subscriptions[s]
					Expect(len(subs)).To(Equal(1))
					Expect(fakeNats.Unsubscriptions).To(ContainElement(subs[0].ID))
				}
			})
		})
	})

	Describe("Request", func() {
		var sid int

		callback := func(msg *yagnats.Message) {}

		BeforeEach(func() {
			sid, _ = nats.Request("subject", []byte("message"), callback)
		})

		It("has a subscription", func() {
			sub := findSubscription(sid, fakeNats.Subscriptions)
			Expect(sub).ShouldNot(BeNil())
			Expect(sub.ID).To(Equal(sid))
		})

		It("creates an inbox", func() {
			sub := findSubscription(sid, fakeNats.Subscriptions)
			Expect(sub.Subject).To(ContainSubstring("_INBOX"))
		})

		It("publishes with a replyTo", func() {
			Expect(len(fakeNats.PublishedMessages)).To(Equal(1))
			for s, msgs := range fakeNats.PublishedMessages {
				Expect(s).To(Equal("subject"))
				Expect(msgs[0].ReplyTo).Should(ContainSubstring("_INBOX"))
			}
		})
	})

})

func findSubscription(sid int, subMap map[string][]yagnats.Subscription) yagnats.Subscription {
	for _, subs := range subMap {
		for _, s := range subs {
			if s.ID == sid {
				return s
			}
		}
	}

	return yagnats.Subscription{}
}

type MockHandler struct {
	handledMessage string
	handledPayload []byte
}

func (m *MockHandler) HandleHealthManagerStart(msg *yagnats.Message) {
}
func (m *MockHandler) HandleRouterStart(msg *yagnats.Message) {
}
func (m *MockHandler) HandleDeaStatus(msg *yagnats.Message) {
}
func (m *MockHandler) HandleDeaDirectedStart(msg *yagnats.Message) {
}
func (m *MockHandler) HandleDeaStop(msg *yagnats.Message) {
}
func (m *MockHandler) HandleDeaUpdate(msg *yagnats.Message) {
}
func (m *MockHandler) HandleDeaFindDroplet(msg *yagnats.Message) {
}
func (m *MockHandler) UUID() string {
	return "UUID"
}
