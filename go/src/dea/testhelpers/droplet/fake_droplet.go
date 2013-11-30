package droplet

import (
	d "dea/droplet"
	"encoding/hex"
	"path"
)

type FakeDropletRegistry struct {
	Droplet *FakeDroplet
}

func (fdr *FakeDropletRegistry) Get(sha1 string) d.Droplet {
	return fdr.Droplet
}

func (fdr *FakeDropletRegistry) Remove(sha1 string) d.Droplet {
	return fdr.Droplet
}

func (fdr *FakeDropletRegistry) Put(sha1 string) {
	fdr.Droplet = &FakeDroplet{Sha1: []byte(sha1)}
}

func (fdr *FakeDropletRegistry) Size() int {
	return 1
}

func (fdr *FakeDropletRegistry) SHA1s() []string {
	return []string{string(fdr.Droplet.SHA1())}
}

type FakeDroplet struct {
	Sha1           []byte
	BaseDir        string
	DropletExists  bool
	DownloadError  error
	LocalCopyError error
}

func NewFakeDroplet(sha1 string) d.Droplet {
	return &FakeDroplet{Sha1: []byte(sha1)}
}

func (fd *FakeDroplet) SHA1() []byte {
	return fd.Sha1
}
func (fd *FakeDroplet) Dir() string {
	return path.Join(fd.BaseDir, hex.EncodeToString(fd.Sha1))
}
func (fd *FakeDroplet) Path() string {
	return path.Join(fd.Dir(), d.DROPLET_BASENAME)
}

func (fd *FakeDroplet) Exists() bool {
	return fd.DropletExists
}

func (fd *FakeDroplet) Download(uri string) error {
	return fd.DownloadError
}

func (fd *FakeDroplet) Local_copy(source string) error {
	return fd.LocalCopyError
}
func (fd *FakeDroplet) Destroy() {
}
