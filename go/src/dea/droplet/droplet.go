package droplet

import (
	"bytes"
	"dea/utils"
	"encoding/hex"
	steno "github.com/cloudfoundry/gosteno"
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
	sha1    []byte
	logger  *steno.Logger
	mutex   sync.Mutex
}

func NewDroplet(baseDir string, sha1 string) (*Droplet, error) {
	shaBytes, err := hex.DecodeString(sha1)
	if err != nil {
		return nil, err
	}
	
	d := &Droplet{
		baseDir: baseDir,
		sha1:    shaBytes,
		logger:  utils.Logger("Droplet", map[string]interface{}{"sha1": sha1}),
		mutex:   sync.Mutex{},
	}

	// Make sure the directory exists
	os.MkdirAll(d.Droplet_dirname(), 0777)

	return d, nil
}

func (d *Droplet) BaseDir() string {
	return d.baseDir
}

func (d *Droplet) Droplet_dirname() string {
	return path.Join(d.baseDir, hex.EncodeToString(d.sha1))
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
	return bytes.Equal(d.sha1, sha1)
}

//func (d *Droplet) SHA1() []byte {
//	return d.sha1
//}

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
		d.logger.Warnf("Failed to create temp file, error: %s", err.Error())
		return err
	}

	err = utils.HttpDownload(uri, download_destination, d.sha1, d.logger)
	if err == nil {
		os.MkdirAll(d.Droplet_dirname(), 0755)
		os.Rename(download_destination.Name(), d.Droplet_path())
		os.Chmod(d.Droplet_path(), 0744)
	}

	return err
}

func (d *Droplet) Local_copy(source string) error {
	d.logger.Debug("Copying local droplet to droplet registry")
	return utils.CopyFile(source, d.Droplet_path())
}

func (d *Droplet) Destroy() {
	dropletName := d.Droplet_dirname()
	dir_to_remove := dropletName + ".deleted." + strconv.FormatInt(time.Now().Unix(), 10)

	// Rename first to both prevent a new instance from referencing a file
	// that is about to be deleted and to avoid doing a potentially expensive
	// operation on the reactor thread.
	d.logger.Debugf("Renaming %s to %s", dropletName, dir_to_remove)
	os.Rename(dropletName, dir_to_remove)

	go func() {
		d.logger.Debugf("Removing %s", dir_to_remove)
		os.RemoveAll(dir_to_remove)
	}()
}
