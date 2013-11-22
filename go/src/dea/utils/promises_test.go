package utils_test

import (
	. "dea/utils"
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sync/atomic"
)

var _ = Describe("Promises", func() {
	Describe("Promise Sequence", func() {
		It("executes promises in sequence", func() {
			cnt := int32(0)
			inc := func() error { atomic.AddInt32(&cnt, 1); return nil }

			err := Sequence_promises(inc, inc, inc)
			Expect(err).To(BeNil())
			Expect(cnt).Should(BeNumerically("==", 3))
		})

		It("fails on first failure", func() {
			cnt := int32(0)
			inc := func() error { atomic.AddInt32(&cnt, 1); return nil }
			fail := func() error { return errors.New("fail") }
			err := Sequence_promises(inc, fail, inc)
			Expect(err).NotTo(BeNil())
			Expect(cnt).Should(BeNumerically("==", 1))

		})

		It("does not invoke promises after a failure", func() {
			cnt := int32(0)
			inc := func() error { atomic.AddInt32(&cnt, 1); return nil }
			fail := func() error { return errors.New("fail") }
			err := Sequence_promises(fail, inc, inc)
			Expect(err).NotTo(BeNil())
			Expect(cnt).Should(BeNumerically("==", 0))
		})

		It("does not fail with no promises", func() {
			Expect(func() { Sequence_promises() }).ToNot(Panic())
		})

		Describe("Panics are converted to errors", func() {
			It("handles recovers from panics", func() {
				panic := func() error { panic("panic!") }
				Expect(func() { Sequence_promises(panic) }).ToNot(Panic())
			})

			It("the error contains the panic'd object as a string", func() {
				panic := func() error { panic(1) }
				var err error
				Expect(func() { err = Sequence_promises(panic) }).ToNot(Panic())
				Expect(err.Error()).To(ContainSubstring("Panic: 1"))
			})

		})
	})

	Describe("Parallel promises", func() {
		It("executes promises in parallel", func() {
			cnt := int32(0)
			inc := func() error { atomic.AddInt32(&cnt, 1); return nil }

			err := Parallel_promises(inc, inc, inc)
			Expect(err).To(BeNil())
			Expect(cnt).Should(BeNumerically("==", 3))
		})

		It("fails on a failure", func() {
			cnt := int32(0)
			inc := func() error { atomic.AddInt32(&cnt, 1); return nil }
			fail := func() error { return errors.New("fail") }
			err := Parallel_promises(inc, fail, inc)
			Expect(err).NotTo(BeNil())
			Expect(cnt).Should(BeNumerically("==", 2))
		})

		It("does not fail with no promises", func() {
			Expect(func() { Parallel_promises() }).ToNot(Panic())
		})

		Describe("Panics are converted to errors", func() {
			It("recovers from panics", func() {
				panic := func() error { panic("panic!") }
				Expect(func() { Parallel_promises(panic) }).ToNot(Panic())
			})

			It("recovers from multiple panics", func() {
				panic := func() error { panic("panic!") }
				Expect(func() { Parallel_promises(panic, panic, panic) }).ToNot(Panic())
			})

			It("the error contains the panic'd object as a string", func() {
				panic := func() error { panic("panic!") }
				var err error
				Expect(func() { err = Parallel_promises(panic) }).ToNot(Panic())
				Expect(err.Error()).To(ContainSubstring("Panic: panic!"))
			})

		})

	})
})
