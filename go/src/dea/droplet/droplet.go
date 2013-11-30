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

const DROPLET_BASENAME = "droplet.tgz"

type Droplet interface {
	SHA1() []byte
	Dir() string
	Path() string
	Exists() bool
	Download(uri string) error
	Local_copy(source string) error
	Destroy()
}

type dribble struct {
	baseDir string
	sha1    []byte
	logger  *steno.Logger
	mutex   sync.Mutex
}

func NewDroplet(baseDir string, sha1 string) (Droplet, error) {
	shaBytes, err := hex.DecodeString(sha1)
	if err != nil {
		return nil, err
	}

	d := &dribble{
		baseDir: baseDir,
		sha1:    shaBytes,
		logger:  utils.Logger("Droplet", map[string]interface{}{"sha1": sha1}),
		mutex:   sync.Mutex{},
	}

	// Make sure the directory exists
	err = os.MkdirAll(d.Dir(), 0777)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dribble) SHA1() []byte {
	return d.sha1
}

func (d *dribble) Dir() string {
	return path.Join(d.baseDir, hex.EncodeToString(d.sha1))
}

func (d *dribble) Path() string {
	return path.Join(d.Dir(), DROPLET_BASENAME)
}

func (d *dribble) Exists() bool {
	_, err := os.Stat(d.Path())
	if os.IsNotExist(err) {
		return false
	}

	sha1, err := utils.SHA1Digest(d.Path())
	return bytes.Equal(d.sha1, sha1)
}

func (d *dribble) Download(uri string) error {
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
	defer os.RemoveAll(download_destination.Name())

	if err = utils.HttpDownload(uri, download_destination, d.sha1, d.logger); err != nil {
		return err
	}

	if err = os.MkdirAll(d.Dir(), 0755); err != nil {
		return err
	}
	if err = os.Rename(download_destination.Name(), d.Path()); err != nil {
		return err
	}
	err = os.Chmod(d.Path(), 0744)

	return err
}

func (d *dribble) Local_copy(source string) error {
	d.logger.Debug("Copying local droplet to droplet registry")
	return utils.CopyFile(source, d.Path())
}

func (d *dribble) Destroy() {
	dropletName := d.Dir()
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
