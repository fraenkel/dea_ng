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

	Describe("Difference", func() {
		It("returns nothing when slices are identical", func() {
			a := []string{"a", "b", "c"}
			b := Difference(a, a)
			Expect(len(b)).To(Equal(0))
		})

		It("returns non-matching items", func() {
			a := []string{"a", "b", "c"}
			b := []string{"d", "b", "e"}
			c := Difference(a, b)
			Expect(len(c)).To(Equal(2))
			Expect(c[0]).To(Equal("a"))
			Expect(c[1]).To(Equal("c"))
		})

		It("empty when nil - slice", func() {
			a := []string{"a", "b", "c"}
			c := Difference(nil, a)
			Expect(len(c)).To(Equal(0))
		})

		It("all items when slice - nil", func() {
			a := []string{"a", "b", "c"}
			c := Difference(nil, a)
			Expect(len(c)).To(Equal(0))

			c = Difference(a, nil)
			Expect(c).To(Equal(a))
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

	Describe("Index", func() {
		It("returns a value when a string is present", func() {
			a := []string{"a", "b", "c"}
			i := Index(a, "b")
			Expect(i).To(Equal(1))
		})

		It("returns -1 when a string is not present", func() {
			a := []string{"a", "b", "c"}
			i := Index(a, "d")
			Expect(i).To(Equal(-1))
		})

		It("returns -1 when the slice is nil", func() {
			i := Index(nil, "d")
			Expect(i).To(Equal(-1))
		})
	})

})
