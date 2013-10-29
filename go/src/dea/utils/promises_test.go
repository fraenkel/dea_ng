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
	})
})
