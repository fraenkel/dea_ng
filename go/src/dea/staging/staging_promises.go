package staging

import (
	"dea/env"
	"dea/utils"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

type StagingPromises interface {
	promise_app_download() error
	promise_buildpack_cache_download() error
	promise_prepare_staging_log() error
	promise_app_dir() error

	promise_unpack_app() error
	promise_unpack_buildpack_cache() error
	promise_stage() error
	promise_pack_app() error
	promise_copy_out() error
	promise_save_droplet() error
	promise_log_upload_started() error
	promise_app_upload() error
	promise_pack_buildpack_cache() error
	promise_copy_out_buildpack_cache() error
	promise_buildpack_cache_upload() error
	promise_staging_info() error
	promise_task_log() error

	promise_save_buildpack_cache() error
}

type stagingPromises struct {
	staging *stagingTask
}

func (p *stagingPromises) promise_log_upload_started() error {
	s := p.staging
	script := "set -o pipefail\n" +
		"droplet_size=`du -h " + s.workspace.warden_staged_droplet() + " | cut -f1`\n" +
		"package_size=`du -h " + s.workspace.downloaded_app_package_path() + " | cut -f1`\n" +
		"echo \"----->Uploading droplet ($droplet_size)\" | tee -a " + s.workspace.warden_staging_log()

	rsp, err := s.Container.RunScript(script)
	s.loggregator_emit_result(rsp)
	return err
}

func (p *stagingPromises) promise_app_upload() error {
	s := p.staging
	uploadUri := s.staging_message.Upload_uri()
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

func (p *stagingPromises) promise_app_download() error {
	s := p.staging
	downloadUri := s.staging_message.Download_uri()
	s.Logger.Infod(map[string]interface{}{"uri": downloadUri},
		"staging.app-download.starting uri")
	download_destination, err := ioutil.TempFile(s.workspace.Tmp_dir(), "app-package-download.tgz")
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

func (p *stagingPromises) promise_buildpack_cache_upload() error {
	s := p.staging
	uploadUri := s.staging_message.Buildpack_cache_upload_uri()
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

func (p *stagingPromises) promise_buildpack_cache_download() error {
	s := p.staging
	downloadUri := s.staging_message.Buildpack_cache_download_uri()
	s.Logger.Infod(map[string]interface{}{
		"uri": downloadUri},
		"staging.buildpack-cache-download.starting")

	download_destination, err := ioutil.TempFile(s.workspace.Tmp_dir(), "buildpack-cache")
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

func (p *stagingPromises) promise_copy_out() error {
	s := p.staging
	src := s.workspace.warden_staged_droplet()
	dst := s.workspace.staged_droplet_dir()
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst},
		"staging.droplet.copying-out")

	return s.Copy_out_request(src, dst)
}

func (p *stagingPromises) promise_save_buildpack_cache() error {
	s := p.staging
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

func (p *stagingPromises) promise_save_droplet() error {
	s := p.staging
	sha1, err := utils.SHA1Digest(s.workspace.staged_droplet_path())
	if err != nil {
		return err
	}

	s.dropletSha1 = string(sha1)

	droplet := s.dropletRegistry.Get(s.dropletSha1)
	if droplet == nil {
		err = fmt.Errorf("Missing droplet")
	} else {
		err = droplet.Local_copy(s.workspace.staged_droplet_path())
	}

	if err != nil {
		s.Logger.Errord(map[string]interface{}{"error": err},
			"staging.droplet.copy-failed")
	}

	return err
}

func (p *stagingPromises) promise_prepare_staging_log() error {
	s := p.staging
	script := fmt.Sprintf("mkdir -p %s/logs && touch %s", s.workspace.warden_staged_dir(),
		s.workspace.warden_staging_log())

	s.Logger.Infod(map[string]interface{}{"script": script},
		"staging.task.preparing-log")

	_, err := s.Container.RunScript(script)
	return err
}

func (p *stagingPromises) promise_app_dir() error {
	s := p.staging

	// Some buildpacks seem to make assumption that /app is a non-empty directory
	// See: https://github.com/heroku/heroku-buildpack-python/blob/master/bin/compile#L46
	// TODO possibly remove this if pull request is accepted
	script := "mkdir -p /app && touch /app/support_heroku_buildpacks && chown -R vcap:vcap /app"

	s.Logger.Infod(map[string]interface{}{"script": script},
		"staging.task.making-app-dir")

	_, err := s.Container.RunScript(script)
	return err
}

func (p *stagingPromises) promise_stage() error {
	s := p.staging
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
		s.workspace.Plugin_config_path(),
		"| tee -a " + s.workspace.warden_staging_log(),
	}, " ")

	s.Logger.Debugd(map[string]interface{}{"script": script},
		"staging.task.execute-staging")

	err = utils.Timeout(s.StagingTimeout()+s.staging_timeout_grace_period(),
		func() error {
			rsp, err := s.Container.RunScript(script)
			s.loggregator_emit_result(rsp)
			return err
		})

	return err
}

func (p *stagingPromises) promise_task_log() error {
	s := p.staging
	src := s.workspace.warden_staging_log()
	dst := path.Dir(s.workspace.staging_log_path())
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst,
	}, "staging.task-log.copying-out")

	return s.Copy_out_request(src, dst)
}

func (p *stagingPromises) promise_staging_info() error {
	s := p.staging
	src := s.workspace.warden_staging_info()
	dest := path.Dir(s.workspace.staging_info_path())
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dest},
		"staging.task-info.copying-out")

	return s.Copy_out_request(src, dest)
}

func (p *stagingPromises) promise_unpack_app() error {
	s := p.staging
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

func (p *stagingPromises) promise_pack_app() error {
	s := p.staging
	s.Logger.Info("staging.task.packing-droplet")

	script := "cd " + s.workspace.warden_staged_dir() + " && " +
		"COPYFILE_DISABLE=true tar -czf " + s.workspace.warden_staged_droplet() + " ."
	_, err := s.Container.RunScript(script)
	return err
}

func (p *stagingPromises) promise_pack_buildpack_cache() error {
	s := p.staging
	// TODO: Ignore if buildpack cache is empty or does not exist

	script := "mkdir -p " + s.workspace.warden_cache() + " && " +
		"cd " + s.workspace.warden_cache() + " && " +
		"COPYFILE_DISABLE=true tar -czf " + s.workspace.warden_staged_buildpack_cache() + " ."
	_, err := s.Container.RunScript(script)
	return err
}

func (p *stagingPromises) promise_unpack_buildpack_cache() error {
	s := p.staging
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

func (p *stagingPromises) promise_copy_out_buildpack_cache() error {
	s := p.staging
	src := s.workspace.warden_staged_buildpack_cache()
	dst := s.workspace.staged_droplet_dir()
	s.Logger.Infod(map[string]interface{}{
		"source":      src,
		"destination": dst},
		"staging.buildpack-cache.copying-out")

	return s.Copy_out_request(src, dst)
}
