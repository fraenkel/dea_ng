package droplet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDroplet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Droplet Suite")
}
