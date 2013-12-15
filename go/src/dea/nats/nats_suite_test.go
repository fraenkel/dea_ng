package nats

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nats Suite")
}
