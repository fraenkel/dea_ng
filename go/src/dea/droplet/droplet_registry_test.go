package droplet

import (
	"encoding/hex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

var _ = Describe("DropletRegistry", func() {
	It("should retrieve droplets that are added", func() {
		tmpDir, _ := ioutil.TempDir("", "dropletregistry")
		defer os.RemoveAll(tmpDir)

		registry := NewDropletRegistry(tmpDir)
		registry.Put("abcdef")
		var d *Droplet = registry.Get("abcdef")
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
			Expect(hex.EncodeToString(droplet.sha1)).To(Equal(sha1))
		}
	})

})
