package droplet

import (
	"dea"
	"os"
	"sync"
)

type dropletRegistry struct {
	baseDir string
	sync.Mutex
	m map[string]dea.Droplet
}

func NewDropletRegistry(baseDir string) dea.DropletRegistry {
	registry := &dropletRegistry{baseDir, sync.Mutex{}, make(map[string]dea.Droplet)}

	// Seed registry with available droplets
	if file, err := os.Open(baseDir); err == nil {
		names, _ := file.Readdirnames(-1)
		file.Close()
		for _, name := range names {
			registry.put(name)
		}
	}

	return registry
}

func (d *dropletRegistry) Get(sha1 string) dea.Droplet {
	d.Lock()
	defer d.Unlock()

	drop := d.m[sha1]
	if drop == nil {
		drop, _ = d.put(sha1)
	}
	return drop
}

func (d *dropletRegistry) Remove(sha1 string) dea.Droplet {
	d.Lock()
	defer d.Unlock()
	if droplet, exists := d.m[sha1]; exists {
		delete(d.m, sha1)
		return droplet
	}
	return nil
}

func (d *dropletRegistry) put(sha1 string) (dea.Droplet, error) {
	droplet, err := NewDroplet(d.baseDir, sha1)
	if err != nil {
		return nil, err
	}

	d.m[sha1] = droplet
	return droplet, nil
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
