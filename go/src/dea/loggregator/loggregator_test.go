package loggregator_test

import (
	. "dea/loggregator"
	temitter "dea/testhelpers/emitter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Loggregator", func() {
	var emitter temitter.FakeEmitter
	var staging_emitter temitter.FakeEmitter

	BeforeEach(func() {
		emitter = temitter.FakeEmitter{}
		SetEmitter(&emitter)
		staging_emitter = temitter.FakeEmitter{}
		SetStagingEmitter(&staging_emitter)
	})

	Describe("dea emitter", func() {
		Describe("emit", func() {
			It("emits to the loggregator", func() {
				Emit("my_app_id", "important log message")
				Expect(emitter.Messages).To(HaveLen(1))
				Expect(emitter.Messages["my_app_id"][0]).To(Equal("important log message"))
			})

			It("does not emit if there is no loggregator", func() {
				SetEmitter(nil)
				Emit("my_app_id", "important log message")
				Expect(emitter.Messages).To(BeNil())
			})

			It("does not emit if there is no application id", func() {
				Emit("", "important log message")
				Expect(emitter.Messages).To(BeNil())
			})
		})

		Describe("emiterror", func() {
			It("emits to the loggregator", func() {
				EmitError("my_app_id", "important log message")
				Expect(emitter.ErrorMessages).To(HaveLen(1))
				Expect(emitter.ErrorMessages["my_app_id"][0]).To(Equal("important log message"))
			})

			It("does not emit if there is no loggregator", func() {
				SetEmitter(nil)
				EmitError("my_app_id", "important log message")
				Expect(emitter.ErrorMessages).To(BeNil())
			})

			It("does not emit if there is no application id", func() {
				EmitError("", "important log message")
				Expect(emitter.ErrorMessages).To(BeNil())
			})
		})
	})

	Describe("staging emitter", func() {
		Describe("emit", func() {
			It("emits to the loggregator", func() {
				StagingEmit("my_app_id", "important log message")
				Expect(staging_emitter.Messages).To(HaveLen(1))
				Expect(staging_emitter.Messages["my_app_id"][0]).To(Equal("important log message"))
			})

			It("does not emit if there is no loggregator", func() {
				SetStagingEmitter(nil)
				StagingEmit("my_app_id", "important log message")
				Expect(staging_emitter.Messages).To(BeNil())
			})

			It("does not emit if there is no application id", func() {
				StagingEmit("", "important log message")
				Expect(staging_emitter.Messages).To(BeNil())
			})
		})

		Describe("emiterror", func() {
			It("emits to the loggregator", func() {
				StagingEmitError("my_app_id", "important log message")
				Expect(staging_emitter.ErrorMessages).To(HaveLen(1))
				Expect(staging_emitter.ErrorMessages["my_app_id"][0]).To(Equal("important log message"))
			})

			It("does not emit if there is no loggregator", func() {
				SetStagingEmitter(nil)
				StagingEmitError("my_app_id", "important log message")
				Expect(staging_emitter.ErrorMessages).To(BeNil())
			})

			It("does not emit if there is no application id", func() {
				StagingEmitError("", "important log message")
				Expect(staging_emitter.ErrorMessages).To(BeNil())
			})
		})
	})
})
