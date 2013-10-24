package staging

import (
	"dea/config"
	"dea/task"
	"dea/utils"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"time"
)

type StagingTask struct {
	id              string
	warden          string
	stagingConfig   *config.StagingConfig
	attributes      map[string]interface{}
	buildpacksInUse []string
	workspace       StagingTaskWorkspace
	task.Task
	dropletSha1             string
	after_setup_callback    func(e error)
	after_complete_callback func(e error)
	after_upload_callback   func(e error)
	after_stop_callback     func(e error)
}

func NewStagingTask(config *config.Config, attributes map[string]interface{}, buildpacksInUse []string) StagingTask {
	s := StagingTask{
		id:              attributes["task_id"].(string),
		attributes:      attributes,
		warden:          config.WardenSocket,
		stagingConfig:   &config.Staging,
		buildpacksInUse: buildpacksInUse,
	}

	envProps := attributes["properties"].(map[string]string)
	s.workspace = NewStagingTaskWorkspace(config.BaseDir, s.AdminBuildpacks(), buildpacksInUse,
		envProps)

	return s
}

func (s StagingTask) Id() string {
	return s.id
}

func (s StagingTask) wardenSocket() string {
	return s.warden
}

func (s StagingTask) StagingTimeout() time.Duration {
	timeout := s.stagingConfig.MaxStagingDuration
	if timeout == nil {
		return 900
	}
	return *timeout
}

func (s StagingTask) MemoryLimit() config.Memory {
	return config.Memory(s.stagingConfig.MemoryLimitMB) * config.Mebi
}

func (s StagingTask) DiskLimit() config.Disk {
	return config.Disk(s.stagingConfig.DiskLimitMB) * config.MB
}

func (s StagingTask) AdminBuildpacks() []string {
	return s.attributes["admin_buildpacks"].([]string)
}

func (s StagingTask) StreamingLogUrl() string {
	// TODO
	panic("Not Implemented")
	//return s.dirServer.StagingTaskFileUrlFor(s.id, workspace.warden_staging_log)
}

func (s StagingTask) task_info() map[string]interface{} {
	data := make(map[string]interface{})
	stagingInfoPath := s.workspace.staging_info_path()
	if _, err := os.Stat(stagingInfoPath); os.IsExist(err) {
		bytes, err := ioutil.ReadFile(stagingInfoPath)
		if err = goyaml.Unmarshal(bytes, &data); err != nil {
			utils.Logger("StagingTask").Errorf("%s", err.Error())
		}
	}

	return data
}

func (s StagingTask) DetectedBuildpack() string {
	return s.task_info()["detected_buildpack"].(string)
}

func (s StagingTask) DropletSHA1() string {
	return s.dropletSha1
}

func (s StagingTask) After_setup_callback(callback func(e error)) {
	s.after_setup_callback = callback
}
func (s StagingTask) After_complete_callback(callback func(e error)) {
	s.after_complete_callback = callback
}
func (s StagingTask) After_upload_callback(callback func(e error)) {
	s.after_upload_callback = callback
}
func (s StagingTask) After_stop_callback(callback func(e error)) {
	s.after_stop_callback = callback
}
