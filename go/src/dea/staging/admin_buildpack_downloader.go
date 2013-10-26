package staging

import (
	"dea/utils"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

type AdminBuildpackDownloader struct {
	buildpacks  []StagingBuildpack
	destination string
}

func NewAdminBuildpackDownloader(buildpacks []StagingBuildpack, destination string) AdminBuildpackDownloader {
	return AdminBuildpackDownloader{buildpacks, destination}
}

func (bpdl AdminBuildpackDownloader) download() {
	utils.Logger("AdminBuildpackDownloader").Debugd(map[string]interface{}{"buildpacks": bpdl.buildpacks},
		"admin-buildpacks.download")

	if len(bpdl.buildpacks) == 0 {
		return
	}

	download_promises := make([]func() error, 0, len(bpdl.buildpacks))
	for _, bp := range bpdl.buildpacks {
		buildpack := bp
		dest := path.Join(bpdl.destination, bp.key)
		if !utils.File_Exists(dest) {
			download_promises = append(download_promises, func() error { return download_buildpack(buildpack, dest) })
		}
	}

	utils.Parallel_promises(download_promises...)
}

func download_buildpack(buildpack StagingBuildpack, dest_dir string) error {
	file, err := ioutil.TempFile("", "temp_admin_buildpack")
	if err != nil {
		utils.Logger("AdminBuildpackDownloader").Errorf("Failed to create tempfile, error: %s", err.Error())
		return err
	}

	defer os.RemoveAll(file.Name())

	err = utils.HttpDownload(buildpack.url.String(), file, "")
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "temp_buildpack_dir")
	if err != nil {
		utils.Logger("AdminBuildpackDownloader").Errorf("Failed to create tempdir, error: %s", err.Error())
		return err
	}

	os.Chmod(tmpDir, 0755)
	cmd := exec.Command("unzip", "-q", file.Name(), "-d", tmpDir)
	err = cmd.Run()
	if err != nil {
		utils.Logger("AdminBuildpackDownloader").Errorf("Failed during (%s %v), error: %s", cmd.Path, cmd.Args, err.Error())
		os.RemoveAll(tmpDir)
		return err
	}

	err = os.Rename(tmpDir, dest_dir)
	if err != nil {
		utils.Logger("AdminBuildpackDownloader").Errorf("Failed during rename (%s %s), error: %s", tmpDir, dest_dir, err.Error())
		os.RemoveAll(tmpDir)
		return err
	}

	return nil
}
