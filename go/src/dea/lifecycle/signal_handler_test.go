package lifecycle

import (
	"dea"
	"dea/protocol"
	tlogger "dea/testhelpers/logger"
	tresponder "dea/testhelpers/responders"
	tstarting "dea/testhelpers/starting"
	"dea/utils"
	"encoding/json"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"syscall"
)

var _ = Describe("SignalHandler", func() {
	var signalHandler SignalHandler
	var fakeresponder *tresponder.FakeResponder
	var fakeevac FakeEvacuationHandler
	var fakeshut FakeShutdownHandler
	var fakenats *fakeyagnats.FakeYagnats
	var fakel tlogger.FakeL
	var iRegistry *tstarting.FakeInstanceRegistry

	BeforeEach(func() {
		fakeresponder = &tresponder.FakeResponder{}
		fakenats = fakeyagnats.New()
		fakel = tlogger.FakeL{}
		fakeevac = FakeEvacuationHandler{}
		fakeshut = FakeShutdownHandler{}

		iRegistry = tstarting.NewFakeInstanceRegistry()
	})

	JustBeforeEach(func() {
		logger := utils.Logger("evac", nil)
		logger.L = &fakel

		signalHandler = NewSignalHandler("id", "ip", []dea.Responder{fakeresponder}, &fakeevac, &fakeshut,
			iRegistry, fakenats, logger)
		signalHandler.Setup()
	})

	AfterEach(func() {
		signalHandler.Stop()
	})

	Describe("trap_term", func() {
		It("shutdowns the system", func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			Eventually(func() bool {
				return fakeshut.ShutdownCalled
			}).Should(BeTrue())
		})
	})

	Describe("trap_int", func() {
		It("shutdowns the system", func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			Eventually(func() bool {
				return fakeshut.ShutdownCalled
			}).Should(BeTrue())
		})
	})

	Describe("trap_quit", func() {
		It("shutdowns the system", func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGQUIT)
			Eventually(func() bool {
				return fakeshut.ShutdownCalled
			}).Should(BeTrue())
		})
	})
	Describe("trap_usr1", func() {
		It("sends a dea shutdown message", func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
			Eventually(func() []yagnats.Message {
				return fakenats.PublishedMessages["dea.shutdown"]
			}).Should(HaveLen(1))

			msg := protocol.GoodbyeMessage{}
			err := json.Unmarshal(fakenats.PublishedMessages["dea.shutdown"][0].Payload, &msg)
			Expect(err).To(BeNil())
			Expect(msg.Id).To(Equal("id"))
			Expect(msg.Ip).To(Equal("ip"))
			Expect(msg.Version).To(Equal(protocol.VERSION))
		})

		It("stops advertising", func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
			Eventually(func() bool {
				return fakeresponder.Stopped
			}).Should(BeTrue())
		})
	})

	Describe("trap_usr2", func() {
		JustBeforeEach(func() {
			syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)
		})

		Context("can shutdown", func() {
			BeforeEach(func() {
				fakeevac.CanShutdown = true
			})

			It("evacuates the system and shuts down", func() {
				Eventually(func() bool {
					return fakeevac.EvacuateCalled
				}).Should(BeTrue())
				Expect(fakeshut.ShutdownCalled).To(BeTrue())
			})

		})

		Context("can't shutdown", func() {
			BeforeEach(func() {
				fakeevac.CanShutdown = false
			})

			It("evacuates the system and doesn't shut down", func() {
				Eventually(func() bool {
					return fakeevac.EvacuateCalled
				}).Should(BeTrue())
				Expect(fakeshut.ShutdownCalled).To(BeFalse())
			})
		})
	})

})

type FakeEvacuationHandler struct {
	EvacuateCalled bool
	CanShutdown    bool
}

func (feh *FakeEvacuationHandler) Evacuate(goodbyeMsg *protocol.GoodbyeMessage) bool {
	feh.EvacuateCalled = true
	return feh.CanShutdown
}

type FakeShutdownHandler struct {
	ShutdownCalled bool
}

func (fsh *FakeShutdownHandler) Shutdown(goodbyeMsg *protocol.GoodbyeMessage) {
	fsh.ShutdownCalled = true
}
