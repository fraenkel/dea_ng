package dea

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDea(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dea Suite")
}
