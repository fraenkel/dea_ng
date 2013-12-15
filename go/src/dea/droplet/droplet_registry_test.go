package droplet

import (
	"dea"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

var _ = Describe("DropletRegistry", func() {
	It("should create droplets that are retrieved", func() {
		tmpDir, _ := ioutil.TempDir("", "dropletregistry")
		defer os.RemoveAll(tmpDir)

		registry := NewDropletRegistry(tmpDir)
		var d dea.Droplet = registry.Get("abcdef")
		Expect(d).NotTo(BeNil())
	})

	It("should initialize with existing droplets", func() {
		tmpDir, _ := ioutil.TempDir("", "dropletregistry")
		defer os.RemoveAll(tmpDir)

		var paths [3]string

		for i := 0; i < 3; i++ {
			paths[i] = path.Join(tmpDir, strconv.Itoa(i)+"0")
		}

		for _, p := range paths {
			os.MkdirAll(p, 0755)
		}
		registry := NewDropletRegistry(tmpDir)
		Expect(registry.Size()).To(Equal(3))

		for _, p := range paths {
			sha1 := path.Base(p)
			droplet := registry.Get(sha1)
			Expect(droplet.SHA1()).To(Equal(sha1))
		}
	})

})
