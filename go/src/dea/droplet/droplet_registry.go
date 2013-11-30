package droplet

import (
	"os"
	"sync"
)

type DropletRegistry interface {
	Get(sha1 string) Droplet
	Remove(sha1 string) Droplet
	Put(sha1 string)
	Size() int
	SHA1s() []string
}

type dropletRegistry struct {
	baseDir string
	sync.Mutex
	m map[string]Droplet
}

func NewDropletRegistry(baseDir string) DropletRegistry {
	registry := &dropletRegistry{baseDir, sync.Mutex{}, make(map[string]Droplet)}

	// Seed registry with available droplets
	if file, err := os.Open(baseDir); err == nil {
		names, _ := file.Readdirnames(-1)
		file.Close()
		for _, name := range names {
			registry.Put(name)
		}
	}

	return registry
}

func (d *dropletRegistry) Get(sha1 string) Droplet {
	d.Lock()
	defer d.Unlock()

	return d.m[sha1]
}

func (d *dropletRegistry) Remove(sha1 string) Droplet {
	d.Lock()
	defer d.Unlock()
	if droplet, exists := d.m[sha1]; exists {
		delete(d.m, sha1)
		return droplet
	}
	return nil
}

func (d *dropletRegistry) Put(sha1 string) {
	d.Lock()
	defer d.Unlock()

	droplet, err := NewDroplet(d.baseDir, sha1)
	if err == nil {
		d.m[sha1] = droplet
	}
}

func (d *dropletRegistry) Size() int {
	d.Lock()
	defer d.Unlock()

	return len(d.m)
}

func (d *dropletRegistry) SHA1s() []string {
	d.Lock()
	defer d.Unlock()
	shas := make([]string, 0, len(d.m))
	for k, _ := range d.m {
		shas = append(shas, k)
	}

	return shas
}
