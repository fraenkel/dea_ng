package staging

import (
	"dea/config"
	cnr "dea/container"
	"dea/droplet"
	"dea/loggregator"
	"dea/task"
	"dea/utils"
	"errors"
	"github.com/cloudfoundry/gordon"
	steno "github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path"
	"strings"
	"time"
)

type Callback func(e error) error

type StagingTaskUrlMaker interface {
	Staging_task_url(task_id, file_path string) string
}

type StagingTask interface {
	Id() string
	StagingMessage() StagingMessage
	StagingConfig() config.StagingConfig
	MemoryLimit() config.Memory
	DiskLimit() config.Disk
	Start() error
	Stop()
	StreamingLogUrl(maker StagingTaskUrlMaker) string
	DetectedBuildpack() string
	DropletSHA1() string
	Path_in_container(pathSuffix string) string
	SetAfter_setup_callback(callback Callback)
	SetAfter_complete_callback(callback Callback)
	SetAfter_stop_callback(callback Callback)
	StagingTimeout() time.Duration
}

type stagingTask struct {
	id              string
	stagingConfig   *config.StagingConfig
	bindMounts      []map[string]string
	staging_message StagingMessage
	workspace       StagingTaskWorkspace
	dropletRegistry droplet.DropletRegistry
	deaRuby         string
	*task.Task
	dropletSha1             string
	after_setup_callback    Callback
	after_complete_callback Callback
	after_stop_callback     Callback
	StagingPromises
	staging_timeout_buffer time.Duration
}

func NewStagingTask(config *config.Config, staging_message StagingMessage, buildpacksInUse []StagingBuildpack,
	dropletRegistry droplet.DropletRegistry, logger *steno.Logger) StagingTask {
	p := &stagingPromises{}
	s := &stagingTask{
		id:                     staging_message.Task_id(),
		bindMounts:             config.BindMounts,
		staging_message:        staging_message,
		stagingConfig:          &config.Staging,
		dropletRegistry:        dropletRegistry,
		deaRuby:                config.DeaRuby,
		Task:                   task.NewTask(config.WardenSocket, logger),
		StagingPromises:        p,
		staging_timeout_buffer: 60 * time.Second,
	}

	p.staging = s

	if s.Logger != nil {
		s.Logger.Set("task_id", s.id)
	}

	s.workspace = NewStagingTaskWorkspace(config.BaseDir, config.BuildpackDir, staging_message, buildpacksInUse)

	return s
}

func (s *stagingTask) Id() string {
	return s.id
}

func (s *stagingTask) StagingConfig() config.StagingConfig {
	return *s.stagingConfig
}

func (s *stagingTask) StagingMessage() StagingMessage {
	return s.staging_message
}

func (s *stagingTask) MemoryLimit() config.Memory {
	stagemem := s.stagingConfig.MemoryLimitMB
	if startmem := s.StagingMessage().StartData().MemoryLimit(); startmem > stagemem {
		stagemem = startmem
	}

	return config.Memory(stagemem) * config.Mebi
}

func (s *stagingTask) DiskLimit() config.Disk {
	stagedisk := s.stagingConfig.DiskLimitMB

	if startdisk := s.StagingMessage().StartData().DiskLimit(); startdisk > stagedisk {
		stagedisk = startdisk
	}

	return config.Disk(stagedisk) * config.MB
}

func (s *stagingTask) Start() error {
	defer func() {
		s.Promise_destroy()
		os.RemoveAll(s.workspace.Workspace_dir())
	}()

	err := s.resolve_staging_setup()
	if err == nil {
		err = s.resolve_staging()
	}

	if err != nil {
		s.Logger.Infof("staging.task.failed: %s", err.Error())
	} else {
		s.Logger.Info("staging.task.completed")
	}

	if err == nil {
		err = s.resolve_staging_upload()
		if err != nil {
			s.Logger.Infof("staging.task.upload-failed: %s", err.Error())
		}
	}

	newErr := s.trigger_after_complete(err)
	if newErr != nil {
		return newErr
	}

	return err
}

func (s *stagingTask) Stop() {
	s.Logger.Info("staging.task.stopped")

	s.SetAfter_complete_callback(nil)
	if s.Container.Handle() != "" {
		s.Promise_stop()
	}

	s.trigger_after_stop(errors.New("StagingTaskStoppedError"))
}

func (s *stagingTask) bind_mounts() []*warden.CreateRequest_BindMount {
	workspaceDirs := []string{s.workspace.Workspace_dir(), s.workspace.buildpack_dir(), s.workspace.Admin_buildpacks_dir()}
	return cnr.CreateBindMounts(workspaceDirs, s.bindMounts)
}

func (s *stagingTask) resolve_staging_setup() error {
	s.workspace.Prepare()
	s.Container.Create(s.bind_mounts(), uint64(s.DiskLimit()), uint64(s.MemoryLimit()), false)

	promises := make([]func() error, 1, 2)
	promises[0] = s.promise_app_download
	if s.staging_message.Buildpack_cache_download_uri() != nil {
		promises = append(promises, s.promise_buildpack_cache_download)
	}

	err := utils.Parallel_promises(promises...)
	if err == nil {
		err = utils.Parallel_promises(
			s.promise_prepare_staging_log,
			s.promise_app_dir,
			s.Container.Update_path_and_ip)
	}

	newErr := s.trigger_after_setup(err)
	if newErr != nil {
		// use the new error if one exists or flow the old one
		err = newErr
	}

	return err
}

func (s *stagingTask) resolve_staging() error {
	defer s.promise_task_log()

	err := utils.Sequence_promises(
		s.promise_unpack_app,
		s.promise_unpack_buildpack_cache,
		s.promise_stage,
		s.promise_pack_app,
		s.promise_copy_out,
		s.promise_save_droplet,
		s.promise_log_upload_started,
		s.promise_staging_info,
	)

	return err
}

func (s *stagingTask) resolve_staging_upload() error {
	return utils.Sequence_promises(
		s.promise_app_upload,
		s.promise_save_buildpack_cache,
	)
}

func (s *stagingTask) StreamingLogUrl(maker StagingTaskUrlMaker) string {
	return maker.Staging_task_url(s.Id(), s.workspace.warden_staging_log())
}

func (s *stagingTask) task_info() map[string]interface{} {
	data := make(map[string]interface{})
	stagingInfoPath := s.workspace.staging_info_path()
	if _, err := os.Stat(stagingInfoPath); !os.IsNotExist(err) {
		bytes, err := ioutil.ReadFile(stagingInfoPath)
		if err = goyaml.Unmarshal(bytes, &data); err != nil {
			s.Logger.Errorf("%s", err.Error())
		}
	}

	return data
}

func (s *stagingTask) DetectedBuildpack() string {
	return s.task_info()["detected_buildpack"].(string)
}

func (s *stagingTask) DropletSHA1() string {
	return s.dropletSha1
}

func (s *stagingTask) Path_in_container(pathSuffix string) string {
	cPath := s.Container.Path()
	if cPath == "" {
		return ""
	}

	// Do not use path.Join since the result is Cleaned
	return strings.Join([]string{cPath, "tmp", "rootfs", pathSuffix}, "/")
}

func (s *stagingTask) SetAfter_setup_callback(callback Callback) {
	s.after_setup_callback = callback
}
func (s *stagingTask) trigger_after_setup(err error) error {
	if s.after_setup_callback != nil {
		return s.after_setup_callback(err)
	}
	return nil
}

func (s *stagingTask) SetAfter_complete_callback(callback Callback) {
	s.after_complete_callback = callback
}

func (s *stagingTask) trigger_after_complete(err error) error {
	if s.after_complete_callback != nil {
		return s.after_complete_callback(err)
	}
	return nil
}

func (s *stagingTask) SetAfter_stop_callback(callback Callback) {
	s.after_stop_callback = callback
}
func (s *stagingTask) trigger_after_stop(err error) error {
	if s.after_stop_callback != nil {
		return s.after_stop_callback(err)
	}
	return nil
}

func (s *stagingTask) run_plugin_path() string {
	return path.Join(s.workspace.buildpack_dir(), "bin", "run")
}

func (s *stagingTask) StagingTimeout() time.Duration {
	return s.stagingConfig.MaxStagingDuration
}

func (s *stagingTask) staging_timeout_grace_period() time.Duration {
	return s.staging_timeout_buffer
}

func (s *stagingTask) loggregator_emit_result(result *warden.RunResponse) *warden.RunResponse {
	if result != nil {
		appId := s.staging_message.App_id()
		loggregator.StagingEmit(appId, result.GetStdout())
		loggregator.StagingEmitError(appId, result.GetStderr())
	}
	return result
}
