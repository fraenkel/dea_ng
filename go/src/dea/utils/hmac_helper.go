package utils

import (
	"crypto/hmac"
	"crypto/sha512"
)

type HMACHelper struct {
	key []byte
}

func NewHMACHelper(key []byte) *HMACHelper {
	return &HMACHelper{
		key: key,
	}
}

func (h *HMACHelper) Create(message string) []byte {
	hmac := hmac.New(sha512.New, h.key)
	return hmac.Sum([]byte(message))
}

func (h *HMACHelper) Compare(correct_hmac []byte, message string) bool {
	generated_hmac := h.Create(message)

	// Use constant_time_compare instead of simple '=='
	// to prevent possible timing attacks.
	// (http://codahale.com/a-lesson-in-timing-attacks/)
	return hmac.Equal(correct_hmac, generated_hmac)

}
