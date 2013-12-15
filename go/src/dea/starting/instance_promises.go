package starting

import (
	"dea"
	cnr "dea/container"
	"dea/env"
	"dea/health_check"
	"dea/utils"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gordon"
	"io/ioutil"
	"path"
	"strings"
	"time"
)

type InstancePromises interface {
	Promise_start() error
	Promise_copy_out() error
	Promise_crash_handler() error
	Promise_container() error
	Promise_droplet() error
	Promise_exec_hook_script(key string) error
	Promise_state(from []dea.State, to dea.State) error
	Promise_extract_droplet() error
	Promise_setup_environment() error
	Link(callback func(error) error)
	Promise_link() (*warden.LinkResponse, error)
	Promise_read_instance_manifest(container_path string) (map[string]interface{}, error)
	Promise_health_check() (bool, error)
}

type instancePromises struct {
	instance *Instance
}

func (ip *instancePromises) Promise_copy_out() error {
	i := ip.instance
	if i.instance_path == "" {
		new_instance_path := path.Join(i.crashesPath, i.Id())
		new_instance_path = path.Clean(new_instance_path)
		if err := i.Copy_out_request("/home/vcap/", new_instance_path); err != nil {
			return err
		}

		i.instance_path = new_instance_path
	}
	return nil
}

func (ip *instancePromises) Promise_crash_handler() error {
	i := ip.instance
	if i.Container.Handle() != "" {
		err := i.Promise_copy_out()
		if err != nil {
			return err
		}
		i.Promise_destroy()

		i.Container.CloseAllConnections()
	}

	return nil
}

func (ip *instancePromises) Promise_container() error {
	i := ip.instance

	dirs := []string{i.droplet().Dir()}
	bindMounts := cnr.CreateBindMounts(dirs, i.bindMounts)

	err := i.Container.Create(bindMounts, uint64(i.DiskLimit()), uint64(i.MemoryLimit()), true)
	if err == nil {
		err = i.Promise_setup_environment()
	}

	return err
}

func (ip *instancePromises) Promise_droplet() (err error) {
	i := ip.instance
	if !i.droplet().Exists() {
		i.Logger.Info("droplet.download.starting")
		start := time.Now()

		err = i.droplet().Download(i.DropletUri())
		msg := "droplet.download.succeeded"
		if err != nil {
			msg = "droplet.download.failed"
		}
		i.Logger.Debugd(map[string]interface{}{"took": time.Since(start)},
			msg)
	} else {
		i.Logger.Info("droplet.download.skipped")
	}

	return err
}

func (ip *instancePromises) Promise_start() error {
	i := ip.instance
	environment := env.NewEnv(NewRunningEnv(i))

	var start_script string
	stagedInfo := i.staged_info()
	if stagedInfo != nil {
		command := i.startData.Start_command
		if command == "" {
			command = stagedInfo["start_command"].(string)
		}
		if command == "" {
			return MissingStartCommand
		}

		sysEnv, err := environment.ExportedSystemEnvironmentVariables()
		if err != nil {
			return err
		}
		start_script = env.NewStartupScriptGenerator(
			command,
			environment.ExportedUserEnvironmentVariables(),
			sysEnv,
		).Generate()
	} else {
		start_script, err := environment.ExportedEnvironmentVariables()
		if err != nil {
			return err
		}
		start_script = start_script + "./startup;\nexit"
	}

	response, err := i.Container.Spawn(start_script, i.FileDescriptorLimit(), NPROC_LIMIT, true)
	if err != nil {
		return err
	}

	i.warden_job_id = response.GetJobId()
	return nil
}

func (ip *instancePromises) Promise_exec_hook_script(key string) error {
	i := ip.instance
	if script_path, exists := i.hooks[key]; exists {
		if utils.File_Exists(script_path) {
			script := make([]string, 0, 10)
			script = append(script, "umask 077")
			envVars, err := env.NewEnv(NewRunningEnv(i)).ExportedEnvironmentVariables()
			if err != nil {
				i.Logger.Warnf("Exception: exec_hook_script hook:%s %s", key, err.Error())
			}
			script = append(script, envVars)
			bytes, err := ioutil.ReadFile(script_path)
			if err != nil {
				return err
			}
			script = append(script, string(bytes))
			script = append(script, "exit")
			_, err = i.Container.RunScript(strings.Join(script, "\n"))
			return err
		} else {
			i.Logger.Warnd(map[string]interface{}{"hook": key, "script_path": "script_path"},
				"droplet.hook-script.missing")
		}
	}
	return nil
}

func (ip *instancePromises) Promise_state(from []dea.State, to dea.State) error {
	i := ip.instance
	for _, s := range from {
		if i.State() == s {
			i.SetState(to)
			return nil
		}
	}
	return errors.New("Cannot transition from " + string(i.State()) + " to " + string(to))
}

func (ip *instancePromises) Promise_extract_droplet() error {
	i := ip.instance
	script := fmt.Sprintf("cd /home/vcap/ && tar zxf %s", i.droplet().Path())
	_, err := i.Container.RunScript(script)
	return err
}

func (ip *instancePromises) Promise_setup_environment() error {
	i := ip.instance
	script := "cd / && mkdir -p home/vcap/app && chown vcap:vcap home/vcap/app && ln -s home/vcap/app /app"
	_, err := i.Container.RunScript(script)
	return err
}

func (ip *instancePromises) Link(callback func(error) error) {
	var response *warden.LinkResponse
	i := ip.instance

	wrapped := func() error {
		var err error
		response, err = i.Promise_link()
		return err
	}

	utils.Async_promise(wrapped, func(err error) error {
		if err != nil {
			i.Logger.Warnd(map[string]interface{}{"error": err},
				"droplet.warden.link.failed")
			i.SetExitStatus(-1, "unknown")
		} else {
			i.SetExitStatus(int64(response.GetExitStatus()), determine_exit_description(response))
			i.Logger.Warnd(map[string]interface{}{"exit_status": i.ExitStatus(),
				"exit_description": i.ExitDescription(),
			}, "droplet.warden.link.completed")
		}

		if err != nil {
			i.Logger.Warnd(map[string]interface{}{"error": err},
				"droplet.link.failed")
		}

		switch i.State() {
		case dea.STATE_STARTING:
			i.SetState(dea.STATE_CRASHED)
		case dea.STATE_RUNNING:
			uptime := time.Now().Sub(i.StateTime(dea.STATE_RUNNING))
			i.Logger.Infod(map[string]interface{}{"uptime": uptime},
				"droplet.instance.uptime")

			i.SetState(dea.STATE_CRASHED)
		default:
			// Linking likely completed because of stop
		}

		if callback != nil {
			err = callback(err)
		}

		if err != nil {
			i.Logger.Errorf("Error occurred during async promise: %s", err.Error())
		}

		return err
	})
}

func (ip *instancePromises) Promise_link() (*warden.LinkResponse, error) {
	i := ip.instance
	rsp, err := i.Container.Link(i.warden_job_id)
	if err == nil {
		i.Logger.Infod(map[string]interface{}{"exit_status": rsp.GetExitStatus()},
			"droplet.warden.link.completed")
	}
	return rsp, err
}

func (ip *instancePromises) Promise_read_instance_manifest(container_path string) (map[string]interface{}, error) {
	if container_path == "" {
		return map[string]interface{}{}, nil
	}

	manifest_path := container_relative_path(container_path, "droplet.yaml")
	manifest := make(map[string]interface{})
	err := utils.Yaml_Load(manifest_path, &manifest)
	return manifest, err
}

func (ip *instancePromises) promise_port_open(port uint32) bool {
	i := ip.instance
	host := i.localIp
	i.Logger.Debugd(map[string]interface{}{"host": host, "port": port},
		"droplet.healthcheck.port")

	callback := newHealthCheckCallback()
	i.healthCheck = health_check.NewPortOpen(host, port, 500*time.Millisecond, callback, i.healthCheckTimeout)
	callback.Wait()
	return *callback.result
}

func (ip *instancePromises) promise_state_file_ready(path string) bool {
	i := ip.instance
	i.Logger.Debugd(map[string]interface{}{"path": path},
		"droplet.healthcheck.file")
	callback := newHealthCheckCallback()
	i.healthCheck = health_check.NewStateFileReady(path, 500*time.Millisecond, callback, 5*time.Minute)
	callback.Wait()
	return *callback.result
}

func (ip *instancePromises) Promise_health_check() (bool, error) {
	i := ip.instance
	i.Logger.Debug("droplet.health-check.get-container-info")
	err := i.Container.Update_path_and_ip()
	if err != nil {
		i.Logger.Errorf("droplet.health-check.container-info-failed: %s", err.Error())
		return false, err

	}
	i.Logger.Debug("droplet.health-check.container-info-ok")

	containerPath := i.Container.Path()
	manifest, err := i.Promise_read_instance_manifest(containerPath)
	if err != nil {
		return false, err
	}

	if manifest["state_file"] != nil {
		manifest_path := container_relative_path(containerPath, manifest["state_file"].(string))
		return ip.promise_state_file_ready(manifest_path), nil
	} else if len(i.ApplicationUris()) > 0 {
		return ip.promise_port_open(i.ContainerPort()), nil
	}
	return true, nil
}
