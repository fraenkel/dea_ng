package dea

import (
	"os"
	"path"
)

type Droplet struct {
	baseDir string
	sha1    string
}

func NewDroplet(baseDir string, sha1 string) *Droplet {
	d := &Droplet{baseDir, sha1}
	
	// Make sure the directory exists
	os.MkdirAll(path.Join(baseDir, sha1), 0777)

	return d
}

func (d *Droplet) BaseDir() string {
	return d.baseDir
}

func (d *Droplet) Sha1() string {
	return d.sha1
}
