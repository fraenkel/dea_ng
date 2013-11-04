package directory_server

import (
	"crypto/hmac"
	"crypto/sha512"
)

type HMACHelper interface {
	Create(message string) []byte
	Compare(correct_hmac []byte, message string) bool
}

type hmacHelper struct {
	key []byte
}

func NewHMACHelper(key []byte) HMACHelper {
	if key == nil {
		return nil
	}

	return &hmacHelper{
		key: key,
	}
}

func (h *hmacHelper) Create(message string) []byte {
	hmac := hmac.New(sha512.New, h.key)
	hmac.Write([]byte(message))
	return hmac.Sum(nil)
}

func (h *hmacHelper) Compare(correct_hmac []byte, message string) bool {
	generated_hmac := h.Create(message)

	// Use constant_time_compare instead of simple '=='
	// to prevent possible timing attacks.
	// (http://codahale.com/a-lesson-in-timing-attacks/)
	return hmac.Equal(correct_hmac, generated_hmac)

}
