package starting

import (
	"dea/config"
	"dea/container"
	"dea/droplet"
	"dea/env"
	"dea/health_check"
	"dea/task"
	"dea/utils"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gordon"
	steno "github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type HealthCheckFailed string
type MissingStartCommand string

func (e HealthCheckFailed) Error() string {
	return "didn't start accepting connections"
}

func (e MissingStartCommand) Error() string {
	return "missing start command"
}

type State string

const (
	STATE_BORN     State = "BORN"
	STATE_STARTING State = "STARTING"
	STATE_RUNNING  State = "RUNNING"
	STATE_STOPPING State = "STOPPING"
	STATE_STOPPED  State = "STOPPED"
	STATE_CRASHED  State = "CRASHED"
	STATE_DELETED  State = "DELETED"
	STATE_RESUMING State = "RESUMING"

	nproc_LIMIT = 512
)

type link_callback func(err error)

type Transition struct {
	From State
	To   State
}

type Instance struct {
	raw_attributes map[string]interface{}
	startData      StartData
	path           *string

	instance_id string
	// private_instance_id is internal id that represents the instance,
	// which is generated by DEA itself. Currently, we broadcast it to
	// all routers. Routers use that as sticky session of the instance.
	private_instance_id string

	instance_path string
	warden_job_id uint32
	state         State
	state_times   map[string]time.Time

	exitStatus      int64
	exitDescription string
	logger          *steno.Logger
	hooks           map[string]string
	statCollector   StatCollector
	task.Task
	utils.EventEmitter
	dropletRegistry    *droplet.DropletRegistry
	stagedInfo         *map[string]interface{}
	localIp            string
	healthCheckTimeout time.Duration
	healthCheck        health_check.HealthCheck
	crashesPath        string
}

type healthcheckCallback struct {
	result    *bool
	condition *sync.Cond
}

func newHealthCheckCallback() *healthcheckCallback {
	mutex := sync.Mutex{}
	cond := sync.NewCond(&mutex)
	return &healthcheckCallback{condition: cond}
}

func (hcc *healthcheckCallback) Success() {
	hcc.condition.L.Lock()
	defer hcc.condition.L.Unlock()

	result := true
	hcc.result = &result
	hcc.condition.Signal()
}

func (hcc *healthcheckCallback) Failure() {
	hcc.condition.L.Lock()
	defer hcc.condition.L.Unlock()

	result := false
	hcc.result = &result
	hcc.condition.Signal()
}

func (hcc *healthcheckCallback) Wait() {
	hcc.condition.Wait()
}

func NewInstance(raw_attributes map[string]interface{}, config *config.Config, dr *droplet.DropletRegistry, localIp string) *Instance {
	sdata := NewStartData(raw_attributes)

	// Generate unique ID
	instance_id := getString(raw_attributes, "instance_id")
	if instance_id == "" {
		instance_id = utils.UUID()
	}

	// Contatenate 2 UUIDs to generate a 32 chars long private_instance_id
	private_id := getString(raw_attributes, "private_instance_id")
	if private_id == "" {
		private_id = utils.UUID() + utils.UUID()
	}

	i := &Instance{
		raw_attributes:      raw_attributes,
		startData:           sdata,
		state:               STATE_BORN,
		state_times:         make(map[string]time.Time),
		instance_id:         instance_id,
		private_instance_id: private_id,
		exitStatus:          -1,
		localIp:             localIp,
		healthCheckTimeout:  config.MaximumHealthCheckTimeout,
		crashesPath:         config.CrashesPath,
		Task:                task.NewTask(config.WardenSocket, nil),
	}

	if config != nil {
		i.hooks = config.Hooks
	}

	i.Logger = utils.Logger("Instance", map[string]interface{}{
		"instance_id":         i.Id(),
		"instance_index":      i.Index(),
		"private_instance_id": i.private_instance_id,
		"application_id":      i.ApplicationId(),
		"application_version": i.ApplicationVersion(),
		"application_name":    i.ApplicationName(),
	})

	warden_handle := getString(raw_attributes, "warden_handle")
	hostPort := getUInt(raw_attributes, "instance_host_port")
	containerPort := getUInt(raw_attributes, "instance_container_port")
	i.Container.Setup(warden_handle, uint32(hostPort), uint32(containerPort))

	return i
}

func (i *Instance) Setup() {
	i.setup_stat_collector()
	i.setup_link()
	i.setup_crash_handler()
}

func (i *Instance) CCPartition() string {
	return i.startData.cc_partition
}

func (i *Instance) SetId() {
	i.instance_id = utils.UUID()
}

func (i *Instance) Id() string {
	return i.instance_id
}

func (i *Instance) Index() int {
	return i.startData.instance_index
}

func (i *Instance) ApplicationId() string {
	return i.startData.application_id
}

func (i *Instance) SetApplicationVersion(version string) {
	i.startData.application_version = version
}

func (i *Instance) ApplicationVersion() string {
	return i.startData.application_version
}

func (i *Instance) ApplicationName() string {
	return i.startData.application_name
}
func (i *Instance) ApplicationUris() []string {
	return i.startData.application_uris
}

func (i *Instance) SetApplicationUris(uris []string) {
	i.startData.application_uris = uris
}

func (i *Instance) DropletSHA1() string {
	return i.startData.droplet_sha1
}
func (i *Instance) DropletUri() string {
	return i.startData.droplet_uri
}
func (i *Instance) droplet() *droplet.Droplet {
	return i.dropletRegistry.Get(i.DropletSHA1())
}

func (i *Instance) StartCommand() string {
	return i.startData.start_command
}
func (i *Instance) Limits() LimitsData {
	return i.startData.limits
}
func (i *Instance) Environment() map[string]string {
	return i.startData.environment
}
func (i *Instance) Services() []ServiceData {
	return i.startData.services
}

func (i *Instance) PrivateInstanceId() string {
	return i.private_instance_id
}

func (i *Instance) MemoryLimit() config.Memory {
	return i.Limits().mem
}

func (i *Instance) DiskLimit() config.Disk {
	return i.Limits().disk
}

func (i *Instance) FileDescriptorLimit() uint64 {
	return i.Limits().fds
}

func (i *Instance) SetState(newState State) {
	transition := Transition{i.State(), newState}

	i.state = newState
	curTime := time.Now()
	i.state_times["current"] = curTime

	state_time := string(newState)
	i.state_times[state_time] = curTime

	i.Emit(transition)
}

func (i *Instance) State() State {
	return i.state
}

func (i *Instance) StateTime(state State) time.Time {
	return i.state_times[string(state)]

}

func (i *Instance) StateTimestamp() time.Time {
	return i.state_times["current"]
}

func (i *Instance) ExitStatus() int32 {
	return int32(i.exitStatus)
}

func (i *Instance) ExitDescription() string {
	return i.exitDescription
}

func (i *Instance) IsAlive() bool {
	switch i.State() {
	case STATE_BORN, STATE_STARTING, STATE_RUNNING, STATE_STOPPING:
		return true
	}
	return false
}

func (i *Instance) IsPathAvailable() bool {
	switch i.State() {
	case STATE_RUNNING, STATE_CRASHED:
		return true
	}
	return false
}

func (i *Instance) Path() (string, error) {
	if i.path == nil {
		if !i.IsPathAvailable() {
			return "", errors.New("Instance path unavailable")
		}
		if i.Container.Path() == "" {
			return "", errors.New("Warden container path not present")
		}

		path := container_relative_path(i.Container.Path())
		i.path = &path
	}

	return *i.path, nil
}

func (i *Instance) attributes_and_stats() map[string]interface{} {
	m := make(map[string]interface{})
	i.startData.MarshalMap(m)

	m["used_memory_in_bytes"] = i.UsedMemory()
	m["used_disk_in_bytes"] = i.UsedDisk()
	m["computed_pcpu"] = i.Computed_pcpu

	return m
}

func (i *Instance) UsedMemory() config.Memory {
	return i.statCollector.UsedMemory
}

func (i *Instance) UsedDisk() config.Disk {
	return i.statCollector.UsedDisk
}

func (i *Instance) Computed_pcpu() float32 {
	return i.statCollector.ComputedPCPU
}

func (i *Instance) promise_copy_out() error {
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

func (i *Instance) setup_crash_handler() {
	// Resuming to crashed state
	i.EventEmitter.On(Transition{STATE_RESUMING, STATE_CRASHED}, func() {
		i.crash_handler()
	})

	i.EventEmitter.On(Transition{STATE_STARTING, STATE_CRASHED}, func() {
		i.crash_handler()
	})

	i.EventEmitter.On(Transition{STATE_RUNNING, STATE_CRASHED}, func() {
		i.crash_handler()
	})
}

func (i *Instance) promise_crash_handler() error {
	if i.Container.Handle() != "" {
		err := i.promise_copy_out()
		if err != nil {
			return err
		}
		i.Promise_destroy()

		i.Container.CloseAllConnections()
	}

	return nil
}

func (i *Instance) crash_handler() {
	err := i.promise_crash_handler()
	if err != nil {
		i.logger.Warnd(map[string]interface{}{"error": err},
			"droplet.crash-handler.error")
	}
}

func (i *Instance) Start(callback func(error)) {
	i.logger.Info("droplet.starting")

	err := i.promise_state([]State{STATE_BORN}, STATE_STARTING)
	if err != nil {
		goto done
	}
	// Concurrently download droplet and setup container
	err = utils.Parallel_promises(
		i.promise_droplet,
		i.promise_container)
	if err != nil {
		goto done
	}

	err = utils.Sequence_promises(i.promise_extract_droplet,
		func() error { return i.promise_exec_hook_script("before_start") },
		i.promise_start)
	if err != nil {
		goto done
	}

	i.On(Transition{STATE_STARTING, STATE_CRASHED}, func() {
		i.cancel_health_check()
	})

	// Fire off link so that the health check can be cancelled when the
	// instance crashes before the health check completes.

	i.link()

	_, err = i.promise_health_check()
	if err != nil {
		goto done
	}

	err = i.promise_state([]State{STATE_STARTING}, STATE_RUNNING)

	if err == nil {
		i.logger.Info("droplet.healthy")
		err = i.promise_exec_hook_script("after_start")
	} else {
		i.logger.Warn("droplet.unhealthy")
		err = HealthCheckFailed("")
	}

done:
	if err != nil {
		// An error occured while starting, mark as crashed
		i.exitDescription = err.Error()
		i.SetState(STATE_CRASHED)
	}

	if callback != nil {
		callback(err)
	}
}

func (i *Instance) promise_container() error {
	bindPath := i.droplet().Droplet_dirname()
	bindMount := warden.CreateRequest_BindMount{SrcPath: &bindPath, DstPath: &bindPath}
	bindMounts := []*warden.CreateRequest_BindMount{&bindMount}

	err := i.Container.Create(bindMounts, uint64(i.DiskLimit()), uint64(i.MemoryLimit()), true)
	if err == nil {
		err = i.promise_setup_environment()
	}

	return err
}

func (i *Instance) Stop() error {
	startTime := time.Now()

	i.logger.Info("droplet.stopping")
	i.promise_exec_hook_script("before_stop")

	err := i.promise_state([]State{STATE_RUNNING, STATE_STARTING}, STATE_STOPPING)
	if err != nil {
		goto done
	}

	err = i.promise_exec_hook_script("after_stop")
	err = i.Promise_stop()
	if err != nil {
		goto done
	}

	err = i.promise_state([]State{STATE_STOPPING}, STATE_STOPPED)

done:
	duration := time.Since(startTime)

	if err != nil {
		// An error occured while starting, mark as crashed
		i.exitDescription = err.Error()
		i.SetState(STATE_CRASHED)
	}

	operation := "stop instance"
	if err != nil {
		i.logger.Warnf("Failed: %s (took %s)", operation, duration)
		i.logger.Warnf("Exception: %s %s", operation, err.Error())
	} else {
		i.logger.Infof("Delivered: %s (took %s)", operation, duration)
	}

	if err != nil {
		// An error occured while starting, mark as crashed
		i.exitDescription = err.Error()
		i.SetState(STATE_CRASHED)
	}

	return err
}

func (i *Instance) promise_droplet() (err error) {
	if !i.droplet().Exists() {
		i.logger.Info("droplet.download.starting")
		start := time.Now()
		err = i.promise_droplet_download()
		i.logger.Infod(map[string]interface{}{"took": time.Since(start)},
			"droplet.download.finished")
	} else {
		i.logger.Info("droplet.download.skipped")
	}

	return err
}

func (i *Instance) promise_start() error {
	environment := env.NewEnv(NewRunningEnv(i))

	var start_script string
	stagedInfo := i.staged_info()
	if stagedInfo != nil {
		command := (*stagedInfo)["start_command"].(string)
		if command == "" {
			return MissingStartCommand("")
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

	response, err := i.Container.Spawn(start_script, i.FileDescriptorLimit(), nproc_LIMIT, true)
	if err != nil {
		return err
	}

	i.warden_job_id = response.GetJobId()
	return nil
}

func (i *Instance) promise_exec_hook_script(key string) error {
	if script_path, exists := i.hooks[key]; exists {
		if utils.File_Exists(script_path) {
			script := make([]string, 0, 10)
			script = append(script, "umask 077")
			envVars, err := env.NewEnv(NewRunningEnv(i)).ExportedEnvironmentVariables()
			if err != nil {
				i.logger.Warnf("Exception: exec_hook_script hook:%s %s", key, err.Error())
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
			i.logger.Warnd(map[string]interface{}{"hook": key, "script_path": "script_path"},
				"droplet.hook-script.missing")
		}
	}
	return nil
}

func (i *Instance) promise_state(from []State, to State) error {
	for _, s := range from {
		if i.State() == s {
			i.SetState(to)
			return nil
		}
	}
	return errors.New("Cannot tranistion from " + string(i.State()) + " to " + string(to))
}

func (i *Instance) promise_extract_droplet() error {
	script := fmt.Sprintf("cd /home/vcap/ && tar zxf %s", i.droplet().Droplet_path())
	_, err := i.Container.RunScript(script)
	return err
}

func (i *Instance) promise_droplet_download() error {
	return i.droplet().Download(i.DropletUri())
}

func (i *Instance) promise_setup_environment() error {
	script := "cd / && mkdir -p home/vcap/app && chown vcap:vcap home/vcap/app && ln -s home/vcap/app /app"
	_, err := i.Container.RunScript(script)
	return err
}

func (i *Instance) setup_stat_collector() {
	i.EventEmitter.On(Transition{STATE_RESUMING, STATE_RUNNING}, func() {
		i.statCollector.start()
	})

	i.EventEmitter.On(Transition{STATE_STARTING, STATE_RUNNING}, func() {
		i.statCollector.start()
	})

	i.EventEmitter.On(Transition{STATE_RUNNING, STATE_STOPPING}, func() {
		i.statCollector.stop()
	})

	i.EventEmitter.On(Transition{STATE_RUNNING, STATE_CRASHED}, func() {
		i.statCollector.stop()
	})
}

func (i *Instance) setup_link() {
	// Resuming to running state
	i.EventEmitter.On(Transition{STATE_RESUMING, STATE_RUNNING}, func() {
		i.link()
	})
}

func (i *Instance) promise_link() (*warden.LinkResponse, error) {
	rsp, err := i.Container.Link(i.warden_job_id)
	if err == nil {
		i.logger.Infod(map[string]interface{}{"exit_status": rsp.GetExitStatus()},
			"droplet.warden.link.completed")
	}
	return rsp, err
}

func (i *Instance) link() {
	response, err := i.promise_link()
	if err != nil {
		i.exitStatus = -1
		i.exitDescription = "unknown"
	} else {
		i.exitStatus = int64(response.GetExitStatus())
		i.exitDescription = determine_exit_description(response)
	}

	switch i.State() {
	case STATE_STARTING:
		i.SetState(STATE_CRASHED)
	case STATE_RUNNING:
		uptime := time.Now().Sub(i.StateTime(STATE_RUNNING))
		i.logger.Infod(map[string]interface{}{"uptime": uptime},
			"droplet.instance.uptime")

		i.SetState(STATE_CRASHED)
	default:
		// Linking likely completed because of stop
	}
}

func (i *Instance) promise_read_instance_manifest(container_path string) (map[string]interface{}, error) {
	if container_path == "" {
		return map[string]interface{}{}, nil
	}

	manifest_path := container_relative_path(container_path, "droplet.yaml")
	manifest := make(map[string]interface{})
	err := utils.Yaml_Load(manifest_path, &manifest)
	return manifest, err
}

func (i *Instance) promise_port_open(port uint32) bool {
	host := i.localIp
	i.logger.Debugd(map[string]interface{}{"host": host, "port": port},
		"droplet.healthcheck.port")

	callback := newHealthCheckCallback()
	i.healthCheck = health_check.NewPortOpen(host, port, 500*time.Millisecond, callback, i.healthCheckTimeout)
	callback.Wait()
	return *callback.result
}

func (i *Instance) promise_state_file_ready(path string) bool {
	i.logger.Debugd(map[string]interface{}{"path": path},
		"droplet.healthcheck.file")
	callback := newHealthCheckCallback()
	i.healthCheck = health_check.NewStateFileReady(path, 500*time.Millisecond, callback, 5*60*time.Second)
	callback.Wait()
	return *callback.result
}

func (i *Instance) cancel_health_check() {
	if i.healthCheck != nil {
		i.healthCheck.Destroy()
		i.healthCheck = nil
	}
}

func (i *Instance) promise_health_check() (bool, error) {
	i.logger.Debug("droplet.health-check.get-container-info")
	err := i.Container.Update_path_and_ip()
	if err != nil {
		i.logger.Errorf("droplet.health-check.container-info-failed: %s", err.Error())
		return false, err

	}
	i.logger.Debug("droplet.health-check.container-info-ok")

	containerPath := i.Container.Path()
	manifest, err := i.promise_read_instance_manifest(containerPath)
	if err != nil {
		return false, err
	}

	if manifest["state_file"] != nil {
		manifest_path := container_relative_path(containerPath, manifest["state_file"].(string))
		return i.promise_state_file_ready(manifest_path), nil
	} else if len(i.ApplicationUris()) > 0 {
		return i.promise_port_open(i.ContainerPort()), nil
	}
	return true, nil
}

func (i Instance) ContainerPort() uint32 {
	return i.Container.NetworkPort(container.CONTAINER_PORT)
}

func (i Instance) HostPort() uint32 {
	return i.Container.NetworkPort(container.HOST_PORT)
}

func (i *Instance) staged_info() *map[string]interface{} {
	if i.stagedInfo == nil {
		tmpdir, err := ioutil.TempDir("", "instance")
		if err != nil {
			i.Logger.Warnf("Failed to create a temporary directory: %s", err.Error())
			return nil
		}
		defer os.RemoveAll(tmpdir)

		staging_file_name := "staging_info.yml"
		copied_file_name := path.Join(tmpdir, staging_file_name)
		i.Copy_out_request(path.Join("/home/vcap", staging_file_name), tmpdir)

		stagedInfo := new(map[string]interface{})
		err = utils.Yaml_Load(copied_file_name, stagedInfo)
		if err != nil {
			return nil
		}
		i.stagedInfo = stagedInfo
	}

	return i.stagedInfo
}

func (i *Instance) Snapshot_attributes() map[string]interface{} {
	sysdrainUrls := make([]string, 0, 1)
	for _, s := range i.Services() {
		drainUrl := s.syslog_drain_url
		if drainUrl != "" {
			sysdrainUrls = append(sysdrainUrls, drainUrl)
		}
	}

	return map[string]interface{}{
		"cc_partition": i.CCPartition(),

		"instance_id":         i.Id(),
		"instance_index":      i.Index(),
		"private_instance_id": i.private_instance_id,

		"warden_handle": i.Container.Handle(),
		"limits":        i.Limits(),

		"environment": i.Environment(),
		"services":    i.Services(),

		"application_id":      i.ApplicationId(),
		"application_version": i.ApplicationVersion(),
		"application_name":    i.ApplicationName(),
		"application_uris":    i.ApplicationUris(),

		"droplet_sha1": i.DropletSHA1(),
		"droplet_uri":  i.DropletUri,

		"start_command": i.StartCommand(),

		"state": i.State(),

		"warden_job_id":           i.warden_job_id,
		"warden_container_path":   i.Container.Path(),
		"warden_host_ip":          i.Container.HostIp(),
		"instance_host_port":      i.HostPort(),
		"instance_container_port": i.ContainerPort(),

		"syslog_drain_urls": sysdrainUrls,

		"state_starting_timestamp": i.StateTime(STATE_STARTING),
	}
}

func container_relative_path(root string, parts ...string) string {
	front := path.Join(root, "tmp", "rootfs", "home", "vcap")
	back := path.Join(parts...)
	return path.Join(front, back)
}

func determine_exit_description(link_response *warden.LinkResponse) string {
	info := link_response.GetInfo()
	if info == nil {
		return "cannot be determined"
	}

	if info.Events != nil && len(info.Events) > 0 {
		return info.Events[0]
	}

	return "app instance exited"
}
