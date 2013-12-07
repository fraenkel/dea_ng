package config_test

import (
	. "dea/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"runtime"
)

var _ = Describe("Config", func() {
	Context("successful load", func() {
		var config Config

		BeforeEach(func() {
			_, file, _, _ := runtime.Caller(0)
			pathToJSON := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
			config, _ = LoadConfig(pathToJSON)
		})

		It("Can load", func() {
			Expect(config).NotTo(BeNil())
		})

		Context("placement_properties in config/dea.yml", func() {
			It("can parse placement properties", func() {
				Expect(config.PlacementProperties["zone"]).To(Equal("zone1"))
			})
		})

	})

	It("Fails when the file does not exist", func() {
		_, err := LoadConfig("/a/b/c")
		Expect(err).NotTo(BeNil())
	})

})
