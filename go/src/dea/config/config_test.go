package config_test

import (
	. "dea/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"runtime"
)

var _ = Describe("Config", func() {
	It("Can load", func() {
		_, file, _, _ := runtime.Caller(0)
		pathToJSON := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
		config, err := ConfigFromFile(pathToJSON)
		Expect(err).To(BeNil())
		Expect(config).NotTo(BeNil())
	})

	It("Fails when the file does not exist", func() {
		config, err := ConfigFromFile("/a/b/c")
		Expect(err).NotTo(BeNil())
		Expect(config).To(BeNil())
	})
})
