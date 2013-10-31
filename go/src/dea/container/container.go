package container

import (
	"dea/utils"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gordon"
	steno "github.com/cloudfoundry/gosteno"
	"runtime"
	"strconv"
)

const (
	HOST_PORT      = "host_port"
	CONTAINER_PORT = "container_port"
)

var logger = utils.Logger("Container", nil)

type Container struct {
	client       *warden.Client
	NetworkPorts map[string]uint32
	handle       string
	path         string
	hostIp       string
}

func NewContainer(wardenSocket string) *Container {
	return New(&warden.ConnectionInfo{SocketPath: wardenSocket})
}

func New(cp warden.ConnectionProvider) *Container {
	client := warden.NewClient(cp)
	if client == nil {
		return nil
	}

	if err := client.Connect(); err != nil {
		logger.Errorf("Connect failed: %s", err.Error())
		return nil
	}

	return &Container{
		client:       client,
		NetworkPorts: make(map[string]uint32),
	}
}

func (c *Container) Setup(handle string, hostPort, containerPort uint32) {
	c.handle = handle
	c.NetworkPorts[HOST_PORT] = hostPort
	c.NetworkPorts[CONTAINER_PORT] = containerPort
}

func (c *Container) RunScript(script string) (*warden.RunResponse, error) {
	response, err := c.client.Run(c.handle, script)
	if err != nil {
		return response, err
	}

	if *response.ExitStatus > 0 {
		exitStatus := strconv.FormatUint(uint64(*response.ExitStatus), 10)
		data := map[string]string{
			"script":      script,
			"exit_status": exitStatus,
		}

		if response.Stdout != nil {
			data["stdout"] = *response.Stdout
		}
		if response.Stderr != nil {
			data["stderr"] = *response.Stderr
		}

		logger := steno.NewLogger("Container")
		logger.Warnf("%s exited with status %s with data %v", script, exitStatus, data)
		return response, errors.New("Script exited with status " + exitStatus)
	}
	return response, nil
}

func (c *Container) Stop() error {
	_, err := c.client.Stop(c.handle)
	return err
}

func (c *Container) Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error {
	err := c.new_container_with_bind_mounts(bind_mounts)
	if err != nil {
		return err
	}

	err = c.limit_disk(disk_limit)
	if err == nil {
		err = c.limit_memory(memory_limit)
		if network && err == nil {
			err = c.setup_network()
		}
	}

	if err != nil {
		c.Destroy()
		return err
	}

	return nil
}

func (c *Container) Destroy() {
	if c.handle == "" {
		bytes := make([]byte, 512)
		runtime.Stack(bytes, false)
		logger.Warnf("Container.destroy.failed: %v %s", c, bytes)
		return
	}

	_, err := c.client.Destroy(c.handle)
	if err != nil {
		logger.Warnf("Error destroying container: %s", err.Error())
	}
	c.handle = ""
}

func (c *Container) CloseAllConnections() {
	c.client.Disconnect()
}

func (c *Container) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {

	resourceLimits := warden.ResourceLimits{Nproc: &nproc_limit,
		Nofile: &file_descriptor_limit}
	req := warden.SpawnRequest{Handle: &c.handle, Script: &script,
		Rlimits:       &resourceLimits,
		DiscardOutput: &discard_output}
	rsp, err := c.client.SpawnWithRequest(&req)
	if err != nil {
		logger.Warnf("Error spawning container: %s", err.Error())
	}

	return rsp, err
}

func (c *Container) Link(jobId uint32) (*warden.LinkResponse, error) {
	rsp, err := c.client.Link(c.handle, jobId)
	if err != nil {
		logger.Warnf("Error linking container: %s", err.Error())
	}

	return rsp, err
}

func (c *Container) CopyOut(sourcePath, destinationPath string, uid int) error {
	_, err := c.client.CopyOut(c.handle, sourcePath, destinationPath, fmt.Sprintf("%d", uid))
	if err != nil {
		logger.Warnf("Error CopyOut container: %s", err.Error())
	}
	return err
}

func (c *Container) new_container_with_bind_mounts(bind_mounts []*warden.CreateRequest_BindMount) error {
	createRequest := warden.CreateRequest{BindMounts: bind_mounts}
	response, err := c.client.CreateByRequest(&createRequest)
	if err != nil {
		return err
	}

	c.handle = response.GetHandle()
	return nil
}

func (c *Container) limit_disk(disk uint64) error {
	disk = disk * 1024 * 1024 // convert to bytes
	_, err := c.client.LimitDisk(c.handle, disk)
	return err
}

func (c *Container) limit_memory(mem uint64) error {
	mem = mem * 1024 * 1024 // convert to bytes
	_, err := c.client.LimitMemory(c.handle, mem)
	return err
}

func (c *Container) setup_network() error {
	netResponse, err := c.client.NetIn(c.handle)
	if err != nil {
		return err
	}

	c.NetworkPorts[HOST_PORT] = netResponse.GetHostPort()
	c.NetworkPorts[CONTAINER_PORT] = netResponse.GetContainerPort()

	return nil
}

func (c *Container) Info() (*warden.InfoResponse, error) {
	if c.handle == "" {
		return nil, errors.New("container handle must not be nil")
	}

	return c.client.Info(c.handle)
}

func (c *Container) Update_path_and_ip() error {
	rsp, err := c.Info()
	if err != nil || rsp.GetContainerPath() == "" {
		return errors.New("container path is not available")
	}

	c.path = rsp.GetContainerPath()
	c.hostIp = rsp.GetHostIp()

	return err
}

func (c *Container) Handle() string {
	return c.handle
}

func (c *Container) Path() string {
	return c.path
}

func (c *Container) HostIp() string {
	return c.hostIp
}
