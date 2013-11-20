package starting

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStarting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Starting Suite")
}
