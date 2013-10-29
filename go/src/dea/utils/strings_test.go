package utils_test

import (
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StringSlice", func() {
	Describe("Intersection", func() {
		It("returns everything when slices are identical", func() {
			a := []string{"a", "b", "c"}
			a = Intersection(a, a)
			Expect(a).To(Equal(a))
		})

		It("returns matching items", func() {
			a := []string{"a", "b", "c"}
			b := []string{"d", "b", "e"}
			c := Intersection(a, b)
			Expect(len(c)).To(Equal(1))
			Expect(c[0]).To(Equal("b"))
		})

		It("empty when a nil slice is passed", func() {
			a := []string{"a", "b", "c"}
			c := Intersection(nil, a)
			Expect(len(c)).To(Equal(0))

			c = Intersection(a, nil)
			Expect(len(c)).To(Equal(0))
		})
	})

	Describe("Contains", func() {
		It("returns true when a string is present", func() {
			a := []string{"a", "b", "c"}
			b := Contains(a, "b")
			Expect(b).To(BeTrue())
		})

		It("returns false when a string is not present", func() {
			a := []string{"a", "b", "c"}
			b := Contains(a, "d")
			Expect(b).To(BeFalse())
		})

		It("returns false when the slice is nil", func() {
			b := Contains(nil, "d")
			Expect(b).To(BeFalse())
		})
	})
})
