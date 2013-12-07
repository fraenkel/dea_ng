package boot_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBoot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boot Suite")
}
