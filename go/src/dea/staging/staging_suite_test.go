package staging

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestStaging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Staging Suite")
}
