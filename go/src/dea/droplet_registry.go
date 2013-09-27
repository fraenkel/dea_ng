package dea

import (
	"os"
	"path/filepath"
)

type DropletRegistry struct {
	baseDir string
	m       map[string]*Droplet
}

func NewDropletRegistry(baseDir string) *DropletRegistry {
	registry := &DropletRegistry{baseDir, make(map[string]*Droplet)}
	baseName := filepath.Base(baseDir)

	// Seed registry with available droplets
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				if info.IsDir() {
					if (info.Name() == baseName) {
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
	return d.m[sha1]
}

func (d *DropletRegistry) Put(sha1 string) {
	d.m[sha1] = NewDroplet(d.baseDir, sha1)
}

func (d *DropletRegistry) Size() int {
	return len(d.m)
}
