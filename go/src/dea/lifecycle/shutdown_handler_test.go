package lifecycle

import (
	"dea"
	"dea/protocol"
	"dea/staging"
	thelpers "dea/testhelpers"
	tds "dea/testhelpers/directory_server"
	tdroplet "dea/testhelpers/droplet"
	tlogger "dea/testhelpers/logger"
	tnats "dea/testhelpers/nats"
	tresponder "dea/testhelpers/responders"
	tstaging "dea/testhelpers/staging"
	tstarting "dea/testhelpers/starting"
	"dea/utils"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("ShutdownHandler", func() {
	var shutdown *shutdownHandler
	var fakenats *tnats.FakeNats
	var fakeresponder *tresponder.FakeResponder
	var fakel tlogger.FakeL
	var pidFile *dea.PidFile
	var fakeDirServer tds.FakeDirectoryServerV2
	var fakeDropletRegistry tdroplet.FakeDropletRegistry
	var fakeInstanceRegistry *tstarting.FakeInstanceRegistry
	var stagingRegistry *staging.StagingTaskRegistry

	goodbyeMessage := protocol.NewGoodbyeMessage("bogus_uuid", "127.0.0.1", nil)

	BeforeEach(func() {
		fakeresponder = &tresponder.FakeResponder{}
		fakel = tlogger.FakeL{}

		fakenats = tnats.NewFakeNats()
		tempfile, _ := ioutil.TempFile("", "shutdown")
		os.Remove(tempfile.Name())
		pidFile, _ = dea.NewPidFile(tempfile.Name())

		fakeDirServer = tds.FakeDirectoryServerV2{}
		fakeDropletRegistry = tdroplet.FakeDropletRegistry{}
		fakeInstanceRegistry = tstarting.NewFakeInstanceRegistry()
		stagingRegistry = staging.NewStagingTaskRegistry(nil, nil, tstaging.FakeStagingTaskCreator)
	})

	AfterEach(func() {
		pidFile.Release()
	})

	JustBeforeEach(func() {
		logger := utils.Logger("shutdown", nil)
		logger.L = &fakel

		shutdown = NewShutdownHandler([]dea.Responder{fakeresponder}, fakeInstanceRegistry,
			stagingRegistry, &fakeDropletRegistry, &fakeDirServer,
			fakenats, nil, pidFile, logger)
	})

	Describe("shutdown", func() {
		It("sends a dea shutdown message", func() {
			shutdown.Shutdown(goodbyeMessage)
			Expect(fakenats.NatsClient.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
			Expect(string(fakenats.NatsClient.PublishedMessages["dea.shutdown"][0].Payload)).To(ContainSubstring("bogus_uuid"))
		})

		It("stops advertising", func() {
			shutdown.Shutdown(goodbyeMessage)
			Expect(fakeresponder.Stopped).To(BeTrue())
		})

		It("unsubscribes the message bus", func() {
			shutdown.Shutdown(goodbyeMessage)
			Expect(fakenats.Unsubscribed).To(BeTrue())
		})

		It("stops the directory server", func() {
			shutdown.Shutdown(goodbyeMessage)
			Expect(fakeDirServer.Stopped).To(BeTrue())
		})

		It("reaps all droplets", func() {
			shutdown.Shutdown(goodbyeMessage)
			Expect(fakeDropletRegistry.Droplet.Destroyed).To(BeTrue())
		})

		Context("with no instances or stagers", func() {
			var terminator FakeTerminator
			JustBeforeEach(func() {
				shutdown.Terminator = &terminator
			})

			It("terminates", func() {
				shutdown.Shutdown(goodbyeMessage)
				Expect(terminator.Terminated).To(BeTrue())
			})
		})

		Context("with instances and/or stagers", func() {
			var instance *tstarting.FakeInstance
			var stager *tstaging.FakeStagingTask

			BeforeEach(func() {
				instance = &tstarting.FakeInstance{}
				fakeInstanceRegistry.Register(instance)

				taskAttrs := thelpers.Valid_staging_attributes()
				stager = &tstaging.FakeStagingTask{
					StagingMsg: staging.NewStagingMessage(taskAttrs),
				}

				stagingRegistry.Register(stager)
			})

			It("stops them", func() {
				shutdown.Shutdown(goodbyeMessage)
				Expect(instance.Stopped).To(BeTrue())
				Expect(stager.Stopped).To(BeTrue())
			})

			Context("when all instances/stagers have stopped successfully", func() {
				var terminator FakeTerminator
				JustBeforeEach(func() {
					shutdown.Terminator = &terminator
				})

				It("terminates", func() {
					shutdown.Shutdown(goodbyeMessage)
					Eventually(func() bool {
						return terminator.Terminated
					}).Should(BeTrue())
				})
			})

			Context("when an stopping fails", func() {
				var terminator FakeTerminator

				BeforeEach(func() {
					instance.StopError = errors.New("Failed to stop.")
				})

				JustBeforeEach(func() {
					shutdown.Terminator = &terminator
					shutdown.Shutdown(goodbyeMessage)
				})

				It("terminates and logs", func() {
					Eventually(func() bool {
						return terminator.Terminated
					}).Should(BeTrue())

					Expect(fakel.Logs["warn"][0]).To(ContainSubstring("Failed to stop."))
				})
			})
		})

		Context("when already called", func() {
			It("does nothing", func() {
				shutdown.Shutdown(goodbyeMessage)
				shutdown.Shutdown(goodbyeMessage)
				Expect(fakenats.NatsClient.PublishedMessages["dea.shutdown"]).To(HaveLen(1))
			})
		})
	})
})

type FakeTerminator struct {
	Terminated bool
}

func (ft *FakeTerminator) Terminate() {
	ft.Terminated = true
}
