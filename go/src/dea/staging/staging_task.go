package staging

import (
	"dea"
	"dea/config"
	cnr "dea/container"
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

type stagingTask struct {
	id              string
	stagingConfig   *config.StagingConfig
	bindMounts      []map[string]string
	staging_message dea.StagingMessage
	workspace       StagingTaskWorkspace
	dropletRegistry dea.DropletRegistry
	deaRuby         string
	*task.Task
	dropletSha1             string
	after_setup_callback    dea.Callback
	after_complete_callback dea.Callback
	after_stop_callback     dea.Callback
	StagingPromises
	staging_timeout_buffer time.Duration
}

func NewStagingTask(config *config.Config, staging_message dea.StagingMessage, buildpacksInUse []dea.StagingBuildpack,
	dropletRegistry dea.DropletRegistry, logger *steno.Logger) dea.StagingTask {
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

func (s *stagingTask) StagingMessage() dea.StagingMessage {
	return s.staging_message
}

func (s *stagingTask) MemoryLimit() config.Memory {
	stagemem := s.stagingConfig.MemoryLimitMB
	if startmem := s.StagingMessage().StartMessage().MemoryLimitMB(); startmem > stagemem {
		stagemem = startmem
	}

	return config.Memory(stagemem) * config.Mebi
}

func (s *stagingTask) DiskLimit() config.Disk {
	stagedisk := s.stagingConfig.DiskLimitMB

	if startdisk := s.StagingMessage().StartMessage().DiskLimitMB(); startdisk > stagedisk {
		stagedisk = startdisk
	}

	return config.Disk(stagedisk) * config.MB
}

func (s *stagingTask) Start() {
	staging_promise := func() error {
		err := s.resolve_staging_setup()
		if err == nil {
			err = s.resolve_staging()
		}

		return err
	}

	utils.Async_promise(staging_promise, func(err error) error {
		defer func() {
			s.Promise_destroy()
			os.RemoveAll(s.workspace.Workspace_dir())
		}()

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
	})
}

func (s *stagingTask) Stop(callback dea.Callback) {
	stopping_promise := func() error {
		s.Logger.Info("staging.task.stopped")

		s.SetAfter_complete_callback(nil)
		if s.Container.Handle() != "" {
			return s.Promise_stop()
		}

		return nil
	}

	utils.Async_promise(stopping_promise, func(e error) error {
		s.trigger_after_stop(errors.New("StagingTaskStoppedError"))
		if callback != nil {
			return callback(e)
		}
		return nil
	})
}

func (s *stagingTask) bind_mounts() []*warden.CreateRequest_BindMount {
	workspaceDirs := []string{s.workspace.Workspace_dir(), s.workspace.buildpack_dir(), s.workspace.Admin_buildpacks_dir()}
	return cnr.CreateBindMounts(workspaceDirs, s.bindMounts)
}

func (s *stagingTask) resolve_staging_setup() error {
	s.workspace.Prepare()
	s.Container.Create(s.bind_mounts(), uint64(s.DiskLimit()), uint64(s.MemoryLimit()), false)

	promises := make([]utils.Promise, 1, 2)
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

func (s *stagingTask) StreamingLogUrl(maker dea.StagingTaskUrlMaker) string {
	return maker.UrlForStagingTask(s.Id(), s.workspace.warden_staging_log())
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

func (s *stagingTask) SetAfter_setup_callback(callback dea.Callback) {
	s.after_setup_callback = callback
}
func (s *stagingTask) trigger_after_setup(err error) error {
	if s.after_setup_callback != nil {
		return s.after_setup_callback(err)
	}
	return nil
}

func (s *stagingTask) SetAfter_complete_callback(callback dea.Callback) {
	s.after_complete_callback = callback
}

func (s *stagingTask) trigger_after_complete(err error) error {
	if s.after_complete_callback != nil {
		return s.after_complete_callback(err)
	}
	return nil
}

func (s *stagingTask) SetAfter_stop_callback(callback dea.Callback) {
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
