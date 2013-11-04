package directory_server

import (
	"encoding/hex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HmacHelper", func() {
	var subject = NewHMACHelper([]byte("key"))
	// echo -n "value" | openssl dgst -sha512 -hmac "key"
	var value_hmac, _ = hex.DecodeString("86951dc765bef95f9474669cd18df7705d99ae47ea3e76a2ca4c22f71656f42ea66e3acdc898c93f475009fa599d0bb83bd5365f36a9cb92c570708f8de5fae8")

	Describe("New", func() {
		It("it returns nil when the key is nil", func() {
			hmacHelper := NewHMACHelper(nil)
			Expect(hmacHelper).To(BeNil())
		})
	})

	Describe("Create", func() {
		It("returns sha1 hmac value", func() {
			hash := subject.Create("value")
			Expect(hash).To(Equal(value_hmac))
		})
	})

	Describe("Compare", func() {
		Context("when string hmac matches given hmac", func() {
			It("returns true", func() {
				Expect(subject.Compare(value_hmac, "value")).To(BeTrue())
			})

		})

		Context("when string hmac does not match given hmac", func() {
			It("returns false", func() {
				Expect(subject.Compare(value_hmac, "value1")).To(BeFalse())
			})

		})
	})
})
