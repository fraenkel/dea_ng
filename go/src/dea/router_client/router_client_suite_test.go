package router_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRouter_client(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router_client Suite")
}
