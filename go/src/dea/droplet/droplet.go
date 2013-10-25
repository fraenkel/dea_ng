package droplet

import (
	"dea/utils"
	"os"
	"path"
	"strconv"
	"time"
)

const droplet_basename = "droplet.tgz"

type Droplet struct {
	baseDir string
	sha1    string
}

func NewDroplet(baseDir string, sha1 string) *Droplet {
	d := &Droplet{baseDir, sha1}

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

func (d *Droplet) Droplet_exists() bool {
	_, err := os.Stat(d.Droplet_path())
	return os.IsExist(err)
}

func (d *Droplet) SHA1() string {
	return d.sha1
}

func (d *Droplet) Download(uri string) error {
	if d.Droplet_exists() {
		return nil
	}

	os.MkdirAll(d.Droplet_dirname(), 0755)

	droplet, err := os.OpenFile(d.Droplet_path(), os.O_CREATE|os.O_WRONLY, 0744)
	if err != nil {
		return err
	}

	err = utils.NewDownload(uri, droplet, d.SHA1()).Download()
	if err == nil {
		utils.Logger("Droplet").Debugf("Downloaded droplet to %s", d.Droplet_path())
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
