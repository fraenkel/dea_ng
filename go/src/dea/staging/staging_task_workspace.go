package staging

import (
	"dea"
	"dea/utils"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path"
)

const (
	DROPLET_FILE         = "droplet.tgz"
	BUILDPACK_CACHE_FILE = "buildpack_cache.tgz"
	STAGING_LOG          = "staging_task.log"
	STAGING_INFO         = "staging_info.yml"
)

var stwLogger = utils.Logger("StagingTaskWorkspace", nil)

type StagingTaskWorkspace struct {
	baseDir               string
	environmentProperties map[string]interface{}
	buildpackManager      BuildpackManager
}

func NewStagingTaskWorkspace(baseDir, system_buildpack_dir string, stagingMsg dea.StagingMessage, buildpacksInUse []dea.StagingBuildpack) StagingTaskWorkspace {
	adminbuildpacks := stagingMsg.AdminBuildpacks()

	buildpackMgr := NewBuildpackManager(path.Join(baseDir, "admin_buildpacks"),
		system_buildpack_dir,
		adminbuildpacks, buildpacksInUse)

	s := StagingTaskWorkspace{
		baseDir:               baseDir,
		environmentProperties: stagingMsg.Properties(),
		buildpackManager:      buildpackMgr,
	}

	os.MkdirAll(s.Workspace_dir(), 0755)
	os.MkdirAll(s.Tmp_dir(), 0755)

	return s
}

func (s StagingTaskWorkspace) write_config_file() error {
	plugin_config := map[string]interface{}{
		"source_dir":        s.warden_unstaged_dir(),
		"dest_dir":          s.warden_staged_dir(),
		"cache_dir":         s.warden_cache(),
		"environment":       s.environmentProperties,
		"staging_info_name": STAGING_INFO,
		"buildpack_dirs":    s.buildpackManager.List(),
	}

	stwLogger.Infod(plugin_config, "write_config_file.starting")

	bytes, err := goyaml.Marshal(plugin_config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(s.Plugin_config_path(), bytes, 0755)

	return err
}

func (s StagingTaskWorkspace) Prepare() {
	s.buildpackManager.Download()
	s.buildpackManager.Clean()
	s.write_config_file()
}

func (s StagingTaskWorkspace) Tmp_dir() string {
	return path.Join(s.baseDir, "tmp")
}

func (s StagingTaskWorkspace) Admin_buildpacks_dir() string {
	return s.buildpackManager.admin_buildpacks_dir()
}

func (s StagingTaskWorkspace) Workspace_dir() string {
	return path.Join(s.baseDir, "staging")
}

func (s StagingTaskWorkspace) buildpack_dir() string {
	return s.buildpackManager.system_buildpacks_dir()
}

func (s StagingTaskWorkspace) staging_info_path() string {
	return path.Join(s.Workspace_dir(), STAGING_INFO)
}

func (s StagingTaskWorkspace) staging_log_path() string {
	return path.Join(s.Workspace_dir(), STAGING_LOG)
}

func (s StagingTaskWorkspace) staged_droplet_dir() string {
	return path.Join(s.Workspace_dir(), "staged")
}

func (s StagingTaskWorkspace) staged_droplet_path() string {
	return path.Join(s.staged_droplet_dir(), DROPLET_FILE)
}

func (s StagingTaskWorkspace) staged_buildpack_cache_path() string {
	return path.Join(s.staged_droplet_dir(), BUILDPACK_CACHE_FILE)
}

func (s StagingTaskWorkspace) downloaded_app_package_path() string {
	return path.Join(s.Workspace_dir(), "app.zip")
}

func (s StagingTaskWorkspace) downloaded_buildpack_cache_path() string {
	return path.Join(s.Workspace_dir(), BUILDPACK_CACHE_FILE)
}

func (s StagingTaskWorkspace) warden_cache() string {
	return "/tmp/cache"
}

func (s StagingTaskWorkspace) warden_unstaged_dir() string {
	return "/tmp/unstaged"
}

func (s StagingTaskWorkspace) warden_staged_dir() string {
	return "/tmp/staged"
}

func (s StagingTaskWorkspace) warden_staged_droplet() string {
	return path.Join("/tmp", DROPLET_FILE)
}

func (s StagingTaskWorkspace) warden_staged_buildpack_cache() string {
	return path.Join("/tmp", BUILDPACK_CACHE_FILE)
}

func (s StagingTaskWorkspace) warden_staging_log() string {
	return path.Join(s.warden_staged_dir(), "logs", STAGING_LOG)
}

func (s StagingTaskWorkspace) warden_staging_info() string {
	return path.Join(s.warden_staged_dir(), STAGING_INFO)
}

func (s StagingTaskWorkspace) Plugin_config_path() string {
	return path.Join(s.Workspace_dir(), "plugin_config")
}
