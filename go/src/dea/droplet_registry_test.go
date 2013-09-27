package dea

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"path"
)

func createTmpDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "droplet_registry")
	if err != nil {
		t.Error(err)
	}
	return tmpDir
}

func TestRetrieveDroplet(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)

	registry := NewDropletRegistry(tmpDir)
	registry.Put("atest")
	var d *Droplet = registry.Get("atest")
	assert.NotNil(t, d)
}

func TestInitializationWithDroplets(t *testing.T) {
	tmpDir := createTmpDir(t)
	defer os.RemoveAll(tmpDir)

	var paths [3]string

	for i := 0; i < 3; i++ {
		paths[i] = path.Join(tmpDir, strconv.Itoa(i))
	}

	for _, p := range paths {
		os.Create(p)
	}
	registry := NewDropletRegistry(tmpDir)
	assert.Equal(t, 3, registry.Size())

	for _, p := range paths {
		sha1 := path.Base(p)
		droplet := registry.Get(sha1)
		assert.Equal(t, sha1, droplet.Sha1())
	}
}
