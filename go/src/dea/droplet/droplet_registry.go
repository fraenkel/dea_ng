package droplet

import (
	"os"
	"path/filepath"
	"sync"
)

type DropletRegistry struct {
	baseDir string
	sync.Mutex
	m map[string]*Droplet
}

func NewDropletRegistry(baseDir string) *DropletRegistry {
	registry := &DropletRegistry{baseDir, sync.Mutex{}, make(map[string]*Droplet)}
	baseName := filepath.Base(baseDir)

	// Seed registry with available droplets
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if info.IsDir() {
				if info.Name() == baseName {
					return nil
				}

				return filepath.SkipDir
			}
			registry.Put(filepath.Base(path))
		}
		return nil
	})

	return registry
}

func (d *DropletRegistry) BaseDir() string {
	return d.baseDir
}

func (d *DropletRegistry) Get(sha1 string) *Droplet {
	d.Lock()
	defer d.Unlock()
	return d.m[sha1]
}

func (d *DropletRegistry) Remove(sha1 string) *Droplet {
	d.Lock()
	defer d.Unlock()
	if droplet, exists := d.m[sha1]; exists {
		delete(d.m, sha1)
		return droplet
	}
	return nil
}

func (d *DropletRegistry) Put(sha1 string) {
	d.Lock()
	defer d.Unlock()

	droplet, err := NewDroplet(d.baseDir, sha1)
	if err == nil {
		d.m[sha1] = droplet
	}
}

func (d *DropletRegistry) Size() int {
	d.Lock()
	defer d.Unlock()

	return len(d.m)
}

func (d *DropletRegistry) SHA1s() []string {
	d.Lock()
	defer d.Unlock()
	shas := make([]string, 0, len(d.m))
	for k, _ := range d.m {
		shas = append(shas, k)
	}

	return shas
}
