package task_test

import (
	cnr "dea/container"
	. "dea/task"
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
		var container cnr.MockContainer
		JustBeforeEach(func() {
			container = cnr.MockContainer{}
			container.Setup("handle", 0, 0)
			task.Container = &container
		})

		It("executes a StopRequest", func() {
			err := task.Promise_stop()
			Expect(err).To(BeNil())
			Expect(container.MStopHandle).To(Equal("handle"))
		})

		It("raises error when the StopRequest fails", func() {
			container.MStopError = errors.New("error")

			err := task.Promise_stop()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("error"))
		})
	})
})
