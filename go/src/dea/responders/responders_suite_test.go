package responders

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestResponders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Responders Suite")
}
