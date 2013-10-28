package staging

import (
	"dea/utils"
	"io/ioutil"
	"os"
	"path"
)

var bpMgrLogger = utils.Logger("BuildpackManager", nil)

type BuildpackManager struct {
	adminBuildpacks_dir  string
	systemBuildpacks_dir string
	admin_buildpacks     []StagingBuildpack
	buildpacks_in_use    []StagingBuildpack
}

func NewBuildpackManager(admin_buildpacks_dir, system_buildpacks_dir string, admin_buildpacks, buildpacks_in_use []StagingBuildpack) BuildpackManager {
	bpMgr := BuildpackManager{
		adminBuildpacks_dir:  admin_buildpacks_dir,
		systemBuildpacks_dir: system_buildpacks_dir,
		admin_buildpacks:     admin_buildpacks,
		buildpacks_in_use:    buildpacks_in_use,
	}

	os.MkdirAll(bpMgr.admin_buildpacks_dir(), 0755)

	return bpMgr
}

func (bpMgr BuildpackManager) admin_buildpacks_dir() string {
	return bpMgr.adminBuildpacks_dir
}

func (bpMgr BuildpackManager) system_buildpacks_dir() string {
	return bpMgr.systemBuildpacks_dir
}

func (bpMgr BuildpackManager) download() {
	NewAdminBuildpackDownloader(bpMgr.admin_buildpacks, bpMgr.adminBuildpacks_dir).download()
}

func (bpMgr BuildpackManager) clean() {
	for _, bp := range bpMgr.buildpacks_needing_deletion() {
		if err := os.RemoveAll(bp); err != nil {
			bpMgrLogger.Errorf("Delete failed for %s, err: %s", bp, err.Error())
		}
	}
}

func (bpMgr BuildpackManager) list() []string {
	paths := bpMgr.admin_buildpacks_in_staging_message()
	paths = append(paths, bpMgr.system_buildpack_paths()...)
	return paths
}

func (bpMgr BuildpackManager) buildpacks_needing_deletion() []string {
	paths := bpMgr.all_buildpack_paths()
	paths = utils.Intersection(paths, bpMgr.admin_buildpacks_in_staging_message())
	paths = utils.Intersection(paths, bpMgr.buildpacks_in_use_paths())

	return paths
}

func (bpMgr BuildpackManager) admin_buildpacks_in_staging_message() []string {
	paths := make([]string, 0, len(bpMgr.admin_buildpacks))
	for _, bp := range bpMgr.admin_buildpacks {
		bpDir := path.Join(bpMgr.adminBuildpacks_dir, bp.key)
		if utils.File_Exists(bpDir) {
			paths = append(paths, bpDir)
		}
	}

	return paths
}

func (bpMgr BuildpackManager) buildpacks_in_use_paths() []string {
	paths := make([]string, 0, len(bpMgr.admin_buildpacks))
	for _, bp := range bpMgr.buildpacks_in_use {
		bpDir := path.Join(bpMgr.adminBuildpacks_dir, bp.key)
		paths = append(paths, bpDir)
	}

	return paths

}

func (bpMgr BuildpackManager) all_buildpack_paths() []string {
	return collectChildrenDirs(bpMgr.adminBuildpacks_dir)
}

func (bpMgr BuildpackManager) system_buildpack_paths() []string {
	return collectChildrenDirs(bpMgr.systemBuildpacks_dir)
}

func collectChildrenDirs(path string) []string {
	children, err := ioutil.ReadDir(path)
	if err != nil {
		bpMgrLogger.Errorf("Enumerating children for %s, error: %s", path, err.Error())
		return []string{}
	}

	paths := make([]string, 0, len(children))
	for _, c := range children {
		paths = append(paths, c.Name())
	}
	return paths
}
