package utils_test

import (
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EventEmitter", func() {
	It("Allows a valid event", func() {
		e := NewEventEmitter()
		e.On("abc", func() {})

		e.On(123, func() {})
	})

	It("Allows multiple subscribers", func() {
		e := NewEventEmitter()
		e.On("abc", func() {})
		e.On("abc", func() {})
	})

	It("Allow nil event", func() {
		e := NewEventEmitter()
		e.On(nil, func() {})
	})

	It("Emit for registered event", func() {
		notified := new(bool)

		e := NewEventEmitter()
		e.On("abc", func() { *notified = true })
		e.Emit("abc")
		Expect(*notified).To(Equal(true))
	})

	It("Emit for unregistered event", func() {
		notified := new(bool)

		e := NewEventEmitter()
		e.On("abc", func() { *notified = true })
		e.Emit(123)
		Expect(*notified).To(Equal(false))
	})

	It("Everyone notified for registered event", func() {
		notified := new(int)

		e := NewEventEmitter()
		e.On("abc", func() { *notified++ })
		e.On("abc", func() { *notified++ })
		e.Emit("abc")
		Expect(*notified).To(Equal(2))
	})
})
