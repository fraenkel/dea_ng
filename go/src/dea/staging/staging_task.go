package staging

import (
	"dea/config"
	"dea/droplet"
	"dea/env"
	"dea/loggregator"
	"dea/task"
	"dea/utils"
	"fmt"
	"github.com/cloudfoundry/gordon"
	steno "github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path"
	"strings"
	"time"
)

type StagingTask struct {
	id              string
	stagingConfig   *config.StagingConfig
	bindMounts      []map[string]string
	staging_message StagingMessage
	buildpacksInUse []StagingBuildpack
	workspace       StagingTaskWorkspace
	dropletRegistry *droplet.DropletRegistry
	deaRuby         string
	stagingTimeout  time.Duration
	task.Task
	dropletSha1             string
	after_setup_callback    func(e error)
	after_complete_callback func(e error)
	after_upload_callback   func(e error)
	after_stop_callback     func(e error)
}

func NewStagingTask(config *config.Config, staging_message StagingMessage,
	buildpacksInUse []StagingBuildpack, dropletRegistry *droplet.DropletRegistry, logger *steno.Logger) *StagingTask {
	s := &StagingTask{
		id:              staging_message.task_id(),
		bindMounts:      config.BindMounts,
		staging_message: staging_message,
		stagingConfig:   &config.Staging,
		buildpacksInUse: buildpacksInUse,
		dropletRegistry: dropletRegistry,
		deaRuby:         config.DeaRuby,
		stagingTimeout:  config.Staging.MaxStagingDuration,
		Task:            task.NewTask(config.WardenSocket, logger),
	}

	if s.Logger != nil {
		s.Logger.Set("task_id", s.id)
	}

	s.workspace = NewStagingTaskWorkspace(config.BaseDir, config.BuildpackDir, staging_message, buildpacksInUse)

	return s
}

func (s *StagingTask) Id() string {
	return s.id
}

func (s *StagingTask) StagingMessage() StagingMessage {
	return s.staging_message
}

func (s *StagingTask) MemoryLimit() config.Memory {
	return config.Memory(s.stagingConfig.MemoryLimitMB) * config.Mebi
}

func (s *StagingTask) DiskLimit() config.Disk {
	return config.Disk(s.stagingConfig.DiskLimitMB) * config.MB
}

func (s *StagingTask) Start() error {
	defer func() {
		s.Promise_destroy()
		os.RemoveAll(s.workspace.workspace_dir())
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

	s.trigger_after_complete(err)

	if err == nil {
		err = s.resolve_staging_upload()
		if err != nil {
			s.Logger.Infof("staging.task.upload-failed: %s", err.Error())
		}
	}

	s.trigger_after_upload(err)

	return err
}

func (s *StagingTask) Stop() {
	s.Logger.Info("staging.task.stopped")

	s.SetAfter_complete_callback(nil)
	var err error
	if s.Container.Handle() != "" {
		err = s.Promise_stop()
	}

	s.trigger_after_stop(err)
}

func (s *StagingTask) bind_mounts() []*warden.CreateRequest_BindMount {
	workspaceDirs := []string{s.workspace.workspace_dir(), s.workspace.buildpack_dir(), s.workspace.admin_buildpacks_dir()}
	mounts := make([]*warden.CreateRequest_BindMount, 0, len(workspaceDirs)+len(s.bindMounts))
	for _, d := range workspaceDirs {
		mounts = append(mounts, &warden.CreateRequest_BindMount{SrcPath: &d, DstPath: &d})
	}

	for _, bindMap := range s.bindMounts {
		srcPath := bindMap["src_path"]
		dstPath, exist := bindMap["dst_path"]
		if !exist {
			dstPath = srcPath
		}

		mounts = append(mounts, &warden.CreateRequest_BindMount{SrcPath: &srcPath, DstPath: &dstPath})
	}

	return mounts
}

func (s *StagingTask) resolve_staging_setup() error {
	s.workspace.prepare()
	s.Container.Create(s.bind_mounts(), uint64(s.DiskLimit()), uint64(s.MemoryLimit()), false)

	promises := make([]func() error, 1, 2)
	promises[0] = s.promise_app_download
	if s.staging_message.buildpack_cache_download_uri() != nil {
		promises = append(promises, s.promise_buildpack_cache_download)
	}

	err := utils.Parallel_promises(promises...)
	if err == nil {
		err = utils.Parallel_promises(
			s.promise_prepare_staging_log,
			s.promise_app_dir,
			s.Container.Update_path_and_ip)
	}

	s.trigger_after_setup(err)
	return err
}

func (s *StagingTask) resolve_staging() error {
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

func (s *StagingTask) resolve_staging_upload() error {
	return utils.Sequence_promises(
		s.promise_app_upload,
		s.promise_save_buildpack_cache,
	)
}

func (s *StagingTask) promise_log_upload_started() error {
	script := "set -o pipefail\n" +
		"droplet_size=`du -h " + s.workspace.warden_staged_droplet() + " | cut -f1`\n" +
		"package_size=`du -h " + s.workspace.downloaded_app_package_path() + " | cut -f1`\n" +
		"echo \"----->Uploading droplet ($droplet_size)\" | tee -a " + s.workspace.warden_staging_log()

	rsp, err := s.Container.RunScript(script)
	s.loggregator_emit_result(rsp)
	return err
}

func (s *StagingTask) promise_app_upload() error {
	uploadUri := s.staging_message.upload_uri()
	s.Logger.Infod(map[string]interface{}{
		"source":      s.workspace.staged_droplet_path(),
		"destination": uploadUri},
		"staging.droplet-upload.starting")

	start := time.Now()
	err := utils.HttpUpload("upload[droplet]", s.workspace.staged_droplet_path(), uploadUri.String(), s.Logger)

	if err != nil {
		s.Logger.Infod(map[string]interface{}{
			"error":       err,
			"duration":    time.Now().Sub(start),
			"destination": uploadUri},
			"staging.task.droplet-upload-failed")
	} else {
		s.Logger.Infod(map[string]interface{}{
			"duration":    time.Now().Sub(start),
			"destination": uploadUri},
			"staging.task.droplet-upload-completed")
	}

	return err
}

func (s *StagingTask) promise_app_download() error {
	downloadUri := s.staging_message.download_uri()
	s.Logger.Infod(map[string]interface{}{"uri": downloadUri},
		"staging.app-download.starting uri")
	download_destination, err := ioutil.TempFile(s.workspace.tmp_dir(), "app-package-download.tgz")
	if err != nil {
		return err
	}

	startTime := time.Now()
	err = utils.HttpDownload(downloadUri.String(), download_destination, nil, s.Logger)
	if err != nil {
		s.Logger.Debugd(
			map[string]interface{}{
				"duration": time.Now().Sub(startTime),
				"uri":      downloadUri,
				"error":    err},
			"staging.app-download.failed")
		return err
	}

	downloadedAppPath := s.workspace.downloaded_app_package_path()
	os.Rename(download_destination.Name(), downloadedAppPath)
	os.Chmod(downloadedAppPath, 0744)

	s.Logger.Debugd(
		map[string]interface{}{
			"duration":    time.Now().Sub(startTime),
			"destination": downloadedAppPath},
		"staging.app-download.completed")

	return nil
}

func (s *StagingTask) promise_buildpack_cache_upload() error {
	uploadUri := s.staging_message.buildpack_cache_upload_uri()
	s.Logger.Infod(map[string]interface{}{
		"source":      s.workspace.staged_buildpack_cache_path(),
		"destination": uploadUri},
		"staging.buildpack-cache-upload.starting")

	start := time.Now()
	err := utils.HttpUpload("upload[buildpack]", s.workspace.staged_buildpack_cache_path(), uploadUri.String(), s.Logger)

	if err != nil {
		s.Logger.Infod(map[string]interface{}{
			"error":       err,
			"duration":    time.Now().Sub(start),
			"destination": uploadUri},
			"staging.task.buildpack-cache-upload-failed")
	} else {
		s.Logger.Infod(map[string]interface{}{
			"duration":    time.Now().Sub(start),
			"destination": uploadUri},
			"staging.task.buildpack-cache-upload-completed")
	}

	return err
}

func (s *StagingTask) promise_buildpack_cache_download() error {
	downloadUri := s.staging_message.buildpack_cache_download_uri()
	s.Logger.Infod(map[string]interface{}{
		"uri": downloadUri},
		"staging.buildpack-cache-download.starting")

	download_destination, err := ioutil.TempFile(s.workspace.tmp_dir(), "buildpack-cache")
	if err != nil {
		return err
	}

	startTime := time.Now()
	err = utils.HttpDownload(downloadUri.String(), download_destination, nil, s.Logger)
	if err != nil {
		s.Logger.Debugd(map[string]interface{}{
			"duration": time.Now().Sub(startTime),
			"uri":      downloadUri,
			"error":    err},
			"staging.buildpack-cache-download.failed")
		return err
	}

	downloadedCachePath := s.workspace.downloaded_buildpack_cache_path()
	os.Rename(download_destination.Name(), downloadedCachePath)
	os.Chmod(downloadedCachePath, 0744)

	s.Logger.Debugd(map[string]interface{}{
		"duration":    time.Now().Sub(startTime),
		"destination": downloadedCachePath},
		"staging.buildpack-cache-download.completed")

	return nil
}

func (s *StagingTask) promise_copy_out() error {
	src := s.workspace.warden_staged_droplet()
	dst := s.workspace.staged_droplet_dir()
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst},
		"staging.droplet.copying-out")

	return s.Copy_out_request(src, dst)
}

func (s *StagingTask) promise_save_buildpack_cache() error {
	start := time.Now()
	err := s.promise_pack_buildpack_cache()
	if err == nil {
		err = utils.Sequence_promises(
			s.promise_copy_out_buildpack_cache,
			s.promise_buildpack_cache_upload)
	}

	if err != nil {
		s.Logger.Warnd(map[string]interface{}{
			"duration": time.Now().Sub(start),
			"error":    err},
			"staging.buildpack-cache.save.failed")
	} else {
		s.Logger.Infod(map[string]interface{}{
			"duration": time.Now().Sub(start)},
			"staging.buildpack-cache.save.completed")
	}

	return err
}

func (s *StagingTask) promise_save_droplet() error {
	sha1, err := utils.SHA1Digest(s.workspace.staged_droplet_path())
	if err != nil {
		return err
	}

	s.dropletSha1 = string(sha1)

	err = s.dropletRegistry.Get(s.dropletSha1).Local_copy(s.workspace.staged_droplet_path())
	if err != nil {
		s.Logger.Errord(map[string]interface{}{"error": err},
			"staging.droplet.copy-failed")

		return err
	}
	return nil
}

func (s *StagingTask) promise_prepare_staging_log() error {
	script := fmt.Sprintf("mkdir -p %s/logs && touch %s", s.workspace.warden_staged_dir(),
		s.workspace.warden_staging_log())

	s.Logger.Infod(map[string]interface{}{"script": script},
		"staging.task.preparing-log")

	_, err := s.Container.RunScript(script)
	return err
}

func (s *StagingTask) promise_app_dir() error {
	// Some buildpacks seem to make assumption that /app is a non-empty directory
	// See: https://github.com/heroku/heroku-buildpack-python/blob/master/bin/compile#L46
	// TODO possibly remove this if pull request is accepted
	script := "mkdir -p /app && touch /app/support_heroku_buildpacks && chown -R vcap:vcap /app"

	s.Logger.Infod(map[string]interface{}{"script": script},
		"staging.task.making-app-dir")

	_, err := s.Container.RunScript(script)
	return err
}

func (s *StagingTask) promise_stage() error {
	env := env.NewEnv(NewStagingEnv(s))
	exportedEnv, err := env.ExportedEnvironmentVariables()
	if err != nil {
		return err
	}

	script := strings.Join([]string{
		"set -o pipefail;",
		exportedEnv,
		s.deaRuby,
		s.run_plugin_path(),
		s.workspace.plugin_config_path(),
		"| tee -a " + s.workspace.warden_staging_log(),
	}, " ")

	s.Logger.Debugd(map[string]interface{}{"script": script},
		"staging.task.execute-staging")

	err = utils.Timeout(s.stagingTimeout+s.staging_timeout_grace_period(),
		func() error {
			rsp, err := s.Container.RunScript(script)
			s.loggregator_emit_result(rsp)
			return err
		})

	return err
}

func (s *StagingTask) promise_task_log() error {
	src := s.workspace.warden_staging_log()
	dst := path.Dir(s.workspace.staging_log_path())
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst,
	}, "staging.task-log.copying-out")

	return s.Copy_out_request(src, dst)
}

func (s *StagingTask) promise_staging_info() error {
	src := s.workspace.warden_staging_info()
	dest := path.Dir(s.workspace.staging_info_path())
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dest},
		"staging.task-info.copying-out")

	return s.Copy_out_request(src, dest)
}

func (s *StagingTask) promise_unpack_app() error {
	dst := s.workspace.warden_unstaged_dir()
	s.Logger.Infod(map[string]interface{}{
		"destination": dst,
	}, "staging.task.unpacking-app")

	script := "set -o pipefail\n" +
		"package_size=`du -h " + s.workspace.downloaded_app_package_path() + " | cut -f1`\n" +
		"echo \"-----> Downloaded app package ($package_size)\" | tee -a " + s.workspace.warden_staging_log() + "\n" +
		"unzip -q " + s.workspace.downloaded_app_package_path() + " -d " + dst

	rsp, err := s.Container.RunScript(script)
	s.loggregator_emit_result(rsp)
	return err
}

func (s *StagingTask) promise_pack_app() error {
	s.Logger.Info("staging.task.packing-droplet")

	script := "cd " + s.workspace.warden_staged_dir() + " && " +
		"COPYFILE_DISABLE=true tar -czf " + s.workspace.warden_staged_droplet() + " ."
	_, err := s.Container.RunScript(script)
	return err
}

func (s *StagingTask) promise_pack_buildpack_cache() error {
	// TODO: Ignore if buildpack cache is empty or does not exist

	script := "mkdir -p " + s.workspace.warden_cache() + " && " +
		"cd " + s.workspace.warden_cache() + " && " +
		"COPYFILE_DISABLE=true tar -czf " + s.workspace.warden_staged_buildpack_cache() + " ."
	_, err := s.Container.RunScript(script)
	return err
}

func (s *StagingTask) promise_unpack_buildpack_cache() error {
	if utils.File_Exists(s.workspace.downloaded_buildpack_cache_path()) {
		s.Logger.Infod(map[string]interface{}{
			"destination": s.workspace.warden_cache(),
		}, "staging.buildpack-cache.unpack")

		script := "set -o pipefail\n" +
			"package_size=`du -h " + s.workspace.downloaded_buildpack_cache_path() + " | cut -f1`\n" +
			"echo \"-----> Downloaded app buildpack cache ($package_size)\" | tee -a " + s.workspace.warden_staging_log() + "\n" +
			"mkdir -p " + s.workspace.warden_cache() + "\n" +
			"tar xfz " + s.workspace.downloaded_buildpack_cache_path() + " -C " + s.workspace.warden_cache()

		rsp, err := s.Container.RunScript(script)
		s.loggregator_emit_result(rsp)
		return err
	}

	return nil
}

func (s *StagingTask) promise_copy_out_buildpack_cache() error {
	src := s.workspace.warden_staged_buildpack_cache()
	dst := s.workspace.staged_droplet_dir()
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst},
		"staging.buildpack-cache.copying-out")

	return s.Copy_out_request(src, dst)
}

func (s *StagingTask) StreamingLogUrl() string {
	// TODO
	panic("Not Implemented")
	//return s.dirServer.StagingTaskFileUrlFor(s.id, workspace.warden_staging_log)
}

func (s *StagingTask) task_info() map[string]interface{} {
	data := make(map[string]interface{})
	stagingInfoPath := s.workspace.staging_info_path()
	if _, err := os.Stat(stagingInfoPath); os.IsExist(err) {
		bytes, err := ioutil.ReadFile(stagingInfoPath)
		if err = goyaml.Unmarshal(bytes, &data); err != nil {
			s.Logger.Errorf("%s", err.Error())
		}
	}

	return data
}

func (s *StagingTask) DetectedBuildpack() string {
	return s.task_info()["detected_buildpack"].(string)
}

func (s *StagingTask) DropletSHA1() string {
	return s.dropletSha1
}

func (s *StagingTask) Path_in_container(pathSuffix string) string {
	cPath := s.Container.Path()
	if cPath == "" {
		return ""
	}

	// Do not use path.Join since the result is Cleaned
	return strings.Join([]string{cPath, "tmp", "rootfs", pathSuffix}, "/")
}

func (s *StagingTask) SetAfter_setup_callback(callback func(e error)) {
	s.after_setup_callback = callback
}
func (s *StagingTask) trigger_after_setup(err error) {
	if s.after_setup_callback != nil {
		s.after_setup_callback(err)
	}
}

func (s *StagingTask) SetAfter_complete_callback(callback func(e error)) {
	s.after_complete_callback = callback
}

func (s *StagingTask) trigger_after_complete(err error) {
	if s.after_complete_callback != nil {
		s.after_complete_callback(err)
	}
}

func (s *StagingTask) SetAfter_upload_callback(callback func(e error)) {
	s.after_upload_callback = callback
}
func (s *StagingTask) trigger_after_upload(err error) {
	if s.after_upload_callback != nil {
		s.after_upload_callback(err)
	}
}

func (s *StagingTask) SetAfter_stop_callback(callback func(e error)) {
	s.after_stop_callback = callback
}
func (s *StagingTask) trigger_after_stop(err error) {
	if s.after_stop_callback != nil {
		s.after_stop_callback(err)
	}
}

func (s *StagingTask) run_plugin_path() string {
	return path.Join(s.workspace.buildpack_dir(), "bin", "run")
}

func (s *StagingTask) StagingTimeout() time.Duration {
	return s.stagingTimeout
}

func (s *StagingTask) staging_timeout_grace_period() time.Duration {
	return 60 * time.Second
}

func (s *StagingTask) loggregator_emit_result(result *warden.RunResponse) *warden.RunResponse {
	if result != nil {
		appId := s.staging_message.app_id()
		loggregator.StagingEmit(appId, result.GetStdout())
		loggregator.StagingEmitError(appId, result.GetStderr())
	}
	return result
}
