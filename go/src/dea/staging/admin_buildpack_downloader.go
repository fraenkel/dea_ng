package staging

import (
	"dea"
	"dea/utils"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

var bpdlLogger = utils.Logger("AdminBuildpackDownloader", nil)

type AdminBuildpackDownloader struct {
	buildpacks  []dea.StagingBuildpack
	destination string
}

func NewAdminBuildpackDownloader(buildpacks []dea.StagingBuildpack, destination string) AdminBuildpackDownloader {
	return AdminBuildpackDownloader{buildpacks, destination}
}

func (bpdl AdminBuildpackDownloader) Download() {
	bpdlLogger.Debugd(map[string]interface{}{"buildpacks": bpdl.buildpacks},
		"admin-buildpacks.download")

	if len(bpdl.buildpacks) == 0 {
		return
	}

	download_promises := make([]utils.Promise, 0, len(bpdl.buildpacks))
	for _, bp := range bpdl.buildpacks {
		buildpack := bp
		dest := path.Join(bpdl.destination, bp.Key)
		if !utils.File_Exists(dest) {
			download_promises = append(download_promises, func() error { return bpdl.download_buildpack(buildpack, dest) })
		}
	}

	utils.Parallel_promises(download_promises...)
}

func (bpdl AdminBuildpackDownloader) download_buildpack(buildpack dea.StagingBuildpack, dest_dir string) error {
	file, err := ioutil.TempFile("", "temp_admin_buildpack")
	if err != nil {
		bpdlLogger.Errorf("Failed to create tempfile, error: %s", err.Error())
		return err
	}

	defer os.RemoveAll(file.Name())

	err = utils.HttpDownload(buildpack.Url.String(), file, "", bpdlLogger)
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "temp_buildpack_dir")
	if err != nil {
		bpdlLogger.Errorf("Failed to create tempdir, error: %s", err.Error())
		return err
	}

	os.Chmod(tmpDir, 0755)
	cmd := exec.Command("unzip", "-q", file.Name(), "-d", tmpDir)
	err = cmd.Run()
	if err != nil {
		bpdlLogger.Errorf("Failed during (%s %v), error: %s", cmd.Path, cmd.Args, err.Error())
		os.RemoveAll(tmpDir)
		return err
	}

	err = os.Rename(tmpDir, dest_dir)
	if err != nil {
		bpdlLogger.Errorf("Failed during rename (%s %s), error: %s", tmpDir, dest_dir, err.Error())
		os.RemoveAll(tmpDir)
		return err
	}

	return nil
}
