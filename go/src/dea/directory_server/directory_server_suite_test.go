package directory_server

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDirectory_server(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Directory_server Suite")
}
