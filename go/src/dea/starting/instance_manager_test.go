package starting_test

import (
	"dea"
	. "dea/starting"
	thelpers "dea/testhelpers"
	tboot "dea/testhelpers/boot"
	tlog "dea/testhelpers/logger"
	trm "dea/testhelpers/resource_manager"
	trtr "dea/testhelpers/router_client"
	tstarting "dea/testhelpers/starting"
	steno "github.com/cloudfoundry/gosteno"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync"
)

var _ = Describe("InstanceManager", func() {

	Describe("creating an instance", func() {
		var fakerm trm.FakeResourceManager
		var fakeboot tboot.FakeBootstrap
		var instanceManager dea.InstanceManager
		var fakesink *tlog.FakeSink
		var fakerouter trtr.FakeRouterClient
		var fakesnap tboot.FakeSnapshot

		BeforeEach(func() {
			fakerm = trm.FakeResourceManager{}
			fakeboot = tboot.FakeBootstrap{
				IRegistry: tstarting.NewFakeInstanceRegistry(),
				RtrClient: &fakerouter,
				ResMgr:    &fakerm,
				Snap:      &fakesnap,
			}

			fakesink = tlog.RegisterFakeSink()
		})

		JustBeforeEach(func() {
			instanceManager = NewInstanceManager(&fakeboot)
		})

		Context("when it successfully validates", func() {
			Context("when there is not enough resources available", func() {
				BeforeEach(func() {
					fakerm.ConstrainedResource = "memory"
				})

				It("logs error indicating not enough resource available", func() {
					instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					record := fakesink.Records[steno.LOG_ERROR][0]
					Expect(record.Message).To(ContainSubstring("instance.start.insufficient-resource"))
					Expect(record.Data["constrained_resource"]).To(Equal("memory"))
				})

				It("marks app as crashed", func() {
					instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					i := fakeboot.SendExitMessageInstance
					Expect(i.State()).To(Equal(dea.STATE_CRASHED))
					Expect(i.ExitDescription()).To(ContainSubstring("memory"))
				})

				It("sends exited message", func() {
					instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					Expect(fakeboot.SendExitMessageReason).To(Equal(EXIT_REASON_CRASHED))
				})

				It("returns nil", func() {
					i := instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					Expect(i).To(BeNil())
				})

			})

			Context("when there is enough resources available", func() {
				It("registers instance in instance registry", func() {
					i := instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					is := fakeboot.IRegistry.Instances()
					Expect(is).To(HaveLen(1))
					Expect(is[0]).To(Equal(i))
				})

				It("returns the new instance", func() {
					i := instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
					Expect(i).ToNot(BeNil())

				})
			})

			Describe("state transitions", func() {
				var instance dea.Instance
				JustBeforeEach(func() {
					instance = instanceManager.CreateInstance(thelpers.Valid_instance_attributes(false))
				})

				Context("when the app is born", func() {
					Context("and it transitions to crashed", func() {

						It("sends exited message with reason: crashed", func() {
							instance.SetState(dea.STATE_CRASHED)
							Expect(fakeboot.SendExitMessageInstance).To(Equal(instance))
							Expect(fakeboot.SendExitMessageReason).To(Equal(EXIT_REASON_CRASHED))
						})
					})
				})

				Context("when the app is starting", func() {
					JustBeforeEach(func() {
						instance.SetState(dea.STATE_STARTING)
					})

					Context("and it transitions to running", func() {
						JustBeforeEach(func() {
							instance.SetState(dea.STATE_RUNNING)
						})

						It("sends heartbeat", func() {
							Expect(fakeboot.SendHeartbeatInvoked).To(BeTrue())
						})

						It("registers with the router", func() {
							Expect(fakerouter.RegisterInstanceInstance).To(Equal(instance))
						})

						It("saves the snapshot", func() {
							Expect(fakesnap.SaveInvoked).To(BeTrue())
						})
					})

					Context("and it transitions to crashed", func() {
						It("sends exited message with reason: crashed", func() {
							instance.SetState(dea.STATE_CRASHED)
							Expect(fakeboot.SendExitMessageInstance).To(Equal(instance))
							Expect(fakeboot.SendExitMessageReason).To(Equal(EXIT_REASON_CRASHED))
						})
					})

					Context("and it transitions to stopping", func() {
						It("sends instance stop message", func() {
							instance.SetState(dea.STATE_STOPPING)
							Expect(fakeboot.SendInstanceStopMessageInstance).To(Equal(instance))
						})
					})
				})
				Context("when the app is running", func() {
					JustBeforeEach(func() {
						instance.SetState(dea.STATE_RUNNING)
					})

					Context("and it transitions to crashed", func() {
						JustBeforeEach(func() {
							instance.SetState(dea.STATE_CRASHED)
						})

						It("sends exited message with reason: crashed", func() {
							Expect(fakeboot.SendExitMessageInstance).To(Equal(instance))
							Expect(fakeboot.SendExitMessageReason).To(Equal(EXIT_REASON_CRASHED))
						})

						It("unregisters with the router", func() {
							Expect(fakerouter.UnregisterInstanceInstance).To(Equal(instance))
						})

						It("saves the snapshot", func() {
							Expect(fakesnap.SaveInvoked).To(BeTrue())
						})

					})

					Context("and it transitions to stopping", func() {
						JustBeforeEach(func() {
							instance.SetState(dea.STATE_STOPPING)
						})

						It("unregisters with the router", func() {
							Expect(fakerouter.UnregisterInstanceInstance).To(Equal(instance))
						})

						It("sends instance stop message", func() {
							Expect(fakeboot.SendInstanceStopMessageInstance).To(Equal(instance))
						})

						It("saves the snapshot", func() {
							Expect(fakesnap.SaveInvoked).To(BeTrue())
						})
					})

					Context("and it transitions to stopping", func() {
						It("sends instance stop message", func() {
							instance.SetState(dea.STATE_STOPPING)
							Expect(fakeboot.SendInstanceStopMessageInstance).To(Equal(instance))
						})
					})
				})

				Context("when the app is stopping", func() {
					var destroyed sync.WaitGroup

					JustBeforeEach(func() {
						destroyed = sync.WaitGroup{}
						hoist := destroyed

						hoist.Add(1)
						p := instance.(*Instance).Task.TaskPromises.(*tstarting.FakePromises)
						p.DestroyCallback = func() {
							hoist.Done()
						}
						instance.SetState(dea.STATE_STOPPING)
					})

					Context("and it transitions to stopped", func() {
						JustBeforeEach(func() {
							instance.SetState(dea.STATE_STOPPED)
						})

						It("unregisters from the instance registry", func() {
							is := fakeboot.IRegistry.Instances()
							Expect(is).To(BeEmpty())
						})

						It("destroys the instance", func() {
							done := false
							go func() {
								destroyed.Wait()
								done = true
							}()

							Eventually(func() bool {
								return done
							}).Should(BeTrue())
						})

					})
				})
			})
		})
	})
})
