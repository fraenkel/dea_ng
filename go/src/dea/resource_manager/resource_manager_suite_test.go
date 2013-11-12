package resource_manager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestResource_manager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource_manager Suite")
}
