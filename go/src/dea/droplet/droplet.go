package droplet

import (
	"dea/utils"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

const droplet_basename = "droplet.tgz"

type Droplet struct {
	baseDir string
	sha1    string
	mutex   sync.Mutex
}

func NewDroplet(baseDir string, sha1 string) *Droplet {
	d := &Droplet{
		baseDir: baseDir,
		sha1:    sha1,
		mutex:   sync.Mutex{},
	}

	// Make sure the directory exists
	os.MkdirAll(d.Droplet_dirname(), 0777)

	return d
}

func (d *Droplet) BaseDir() string {
	return d.baseDir
}

func (d *Droplet) Droplet_dirname() string {
	return path.Join(d.baseDir, d.sha1)
}

func (d *Droplet) Droplet_path() string {
	return path.Join(d.Droplet_dirname(), droplet_basename)
}

func (d *Droplet) Exists() bool {
	_, err := os.Stat(d.Droplet_path())
	if os.IsNotExist(err) {
		return false
	}

	sha1, err := utils.SHA1Digest(d.Droplet_path())
	return d.sha1 == string(sha1)
}

func (d *Droplet) SHA1() string {
	return d.sha1
}

func (d *Droplet) Download(uri string) error {
	// ensure only one download is happening for a single droplet.
	// this keeps 100 starts from causing a network storm.

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.Exists() {
		return nil
	}

	download_destination, err := ioutil.TempFile("", "droplet-download.tgz")
	if err != nil {
		utils.Logger("Droplet").Warnf("Failed to create temp file, error: %s", err.Error())
		return err
	}

	err = utils.HttpDownload(uri, download_destination, d.SHA1())
	if err == nil {
		os.MkdirAll(d.Droplet_dirname(), 0755)
		os.Rename(download_destination.Name(), d.Droplet_path())
		os.Chmod(d.Droplet_path(), 0744)
	}

	return err
}

func (d *Droplet) Local_copy(source string) error {
	utils.Logger("Droplet").Debug("Copying local droplet to droplet registry")
	return utils.CopyFile(source, d.Droplet_path())
}

func (d *Droplet) Destroy() {
	dropletName := d.Droplet_dirname()
	dir_to_remove := dropletName + ".deleted." + strconv.FormatInt(time.Now().Unix(), 10)

	// Rename first to both prevent a new instance from referencing a file
	// that is about to be deleted and to avoid doing a potentially expensive
	// operation on the reactor thread.
	utils.Logger("Droplet").Debugf("Renaming %s to %s", dropletName, dir_to_remove)
	os.Rename(dropletName, dir_to_remove)

	go func() {
		utils.Logger("Droplet").Debugf("Removing %s", dir_to_remove)
		os.RemoveAll(dir_to_remove)
	}()
}
