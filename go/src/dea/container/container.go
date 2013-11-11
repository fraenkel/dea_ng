package container

import (
	"dea/utils"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gordon"
	"runtime"
	"strconv"
)

const (
	HOST_PORT      = "host_port"
	CONTAINER_PORT = "container_port"
)

var logger = utils.Logger("sContainer", nil)

type Container interface {
	Setup(handle string, hostPort, containerPort uint32)
	Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error
	Stop() error
	CloseAllConnections()
	Destroy()
	RunScript(script string) (*warden.RunResponse, error)
	Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error)
	Link(jobId uint32) (*warden.LinkResponse, error)
	CopyOut(sourcePath, destinationPath string, uid int) error
	Info() (*warden.InfoResponse, error)
	Update_path_and_ip() error
	Handle() string
	Path() string
	HostIp() string
	NetworkPort(string) uint32
}

type sContainer struct {
	connected    bool
	client       *warden.Client
	networkPorts map[string]uint32
	handle       string
	path         string
	hostIp       string
}

func NewContainer(wardenSocket string) Container {
	return New(&warden.ConnectionInfo{SocketPath: wardenSocket})
}

func New(cp warden.ConnectionProvider) Container {
	client := warden.NewClient(cp)
	if client == nil {
		return nil
	}

	return &sContainer{
		client:       client,
		networkPorts: make(map[string]uint32),
	}
}

func (c *sContainer) Setup(handle string, hostPort, containerPort uint32) {
	c.handle = handle

	if hostPort != 0 {
		c.networkPorts[HOST_PORT] = hostPort
	}
	if containerPort != 0 {
		c.networkPorts[CONTAINER_PORT] = containerPort
	}
}

func (c *sContainer) connect() error {
	if !c.connected {
		err := c.client.Connect()
		if err != nil {
			return err
		}
		c.connected = true
	}

	return nil
}

func (c *sContainer) RunScript(script string) (*warden.RunResponse, error) {
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

		logger.Warnf("%s exited with status %s with data %v", script, exitStatus, data)
		return response, errors.New("Script exited with status " + exitStatus)
	}
	return response, nil
}

func (c *sContainer) Stop() error {
	c.connect()
	_, err := c.client.Stop(c.handle)
	return err
}

func (c *sContainer) Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error {
	c.connect()
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

func (c *sContainer) Destroy() {
	if c.handle == "" {
		bytes := make([]byte, 512)
		runtime.Stack(bytes, false)
		logger.Warnf("sContainer.destroy.failed: %v %s", c, bytes)
		return
	}

	c.connect()
	_, err := c.client.Destroy(c.handle)
	if err != nil {
		logger.Warnf("Error destroying container: %s", err.Error())
	}
	c.handle = ""
}

func (c *sContainer) CloseAllConnections() {
	c.client.Disconnect()
}

func (c *sContainer) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {
	c.connect()

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

func (c *sContainer) Link(jobId uint32) (*warden.LinkResponse, error) {
	c.connect()
	rsp, err := c.client.Link(c.handle, jobId)
	if err != nil {
		logger.Warnf("Error linking container: %s", err.Error())
	}

	return rsp, err
}

func (c *sContainer) CopyOut(sourcePath, destinationPath string, uid int) error {
	c.connect()
	_, err := c.client.CopyOut(c.handle, sourcePath, destinationPath, fmt.Sprintf("%d", uid))
	if err != nil {
		logger.Warnf("Error CopyOut container: %s", err.Error())
	}
	return err
}

func (c *sContainer) new_container_with_bind_mounts(bind_mounts []*warden.CreateRequest_BindMount) error {
	c.connect()

	createRequest := warden.CreateRequest{BindMounts: bind_mounts}
	response, err := c.client.CreateByRequest(&createRequest)
	if err != nil {
		return err
	}

	c.handle = response.GetHandle()
	return nil
}

func (c *sContainer) limit_disk(disk uint64) error {
	c.connect()

	disk = disk * 1024 * 1024 // convert to bytes
	_, err := c.client.LimitDisk(c.handle, disk)
	return err
}

func (c *sContainer) limit_memory(mem uint64) error {
	c.connect()

	mem = mem * 1024 * 1024 // convert to bytes
	_, err := c.client.LimitMemory(c.handle, mem)
	return err
}

func (c *sContainer) setup_network() error {
	c.connect()

	netResponse, err := c.client.NetIn(c.handle)
	if err != nil {
		return err
	}

	c.networkPorts[HOST_PORT] = netResponse.GetHostPort()
	c.networkPorts[CONTAINER_PORT] = netResponse.GetContainerPort()

	return nil
}

func (c *sContainer) Info() (*warden.InfoResponse, error) {
	if c.handle == "" {
		return nil, errors.New("container handle must not be nil")
	}

	return c.client.Info(c.handle)
}

func (c *sContainer) Update_path_and_ip() error {
	rsp, err := c.Info()
	if err != nil || rsp.GetContainerPath() == "" {
		return errors.New("container path is not available")
	}

	c.path = rsp.GetContainerPath()
	c.hostIp = rsp.GetHostIp()

	return err
}

func (c *sContainer) Handle() string {
	return c.handle
}

func (c *sContainer) Path() string {
	return c.path
}

func (c *sContainer) HostIp() string {
	return c.hostIp
}

func (c *sContainer) NetworkPort(portName string) uint32 {
	return c.networkPorts[portName]
}
