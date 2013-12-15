package task_test

import (
	. "dea/task"
	tcnr "dea/testhelpers/container"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Task", func() {
	var task *Task

	JustBeforeEach(func() {
		task = NewTask("some socket", nil)
	})

	Describe("container", func() {
		It("creates a container with connection provider", func() {
			Expect(task.Container).ToNot(BeNil())
		})
	})

	Describe("Promise_stop", func() {
		var container tcnr.FakeContainer
		JustBeforeEach(func() {
			container = tcnr.FakeContainer{}
			container.Setup("handle", 0, 0)
			task.Container = &container
		})

		It("executes a StopRequest", func() {
			err := task.Promise_stop()
			Expect(err).To(BeNil())
			Expect(container.FStopHandle).To(Equal("handle"))
		})

		It("raises error when the StopRequest fails", func() {
			container.FStopError = errors.New("error")

			err := task.Promise_stop()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error"))
		})
	})
})
