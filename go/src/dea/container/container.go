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

type Container struct {
	client       *warden.Client
	NetworkPorts map[string]uint32
	handle       string
	path         string
	hostIp       string
}

func NewContainer(wardenSocket string) Container {
	return Container{
		client: warden.NewClient(&warden.ConnectionInfo{SocketPath: wardenSocket}),
	}
}

func (c *Container) RunScript(script string) (*warden.RunResponse, error) {
	response, err := c.client.Run(c.handle, script)
	if err != nil {
		return nil, err
	}

	if *response.ExitStatus > 0 {
		exitStatus := strconv.FormatUint(uint64(*response.ExitStatus), 10)
		data := map[string]string{
			"script":      script,
			"exit_status": exitStatus,
			"stdout":      *response.Stdout,
			"stderr":      *response.Stderr,
		}

		logger := steno.NewLogger("Container")
		logger.Warnf("%s exited with status %s with data %v", script, exitStatus, data)
		return nil, errors.New("Script exited with status " + exitStatus)
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
		utils.Logger("Container").Warnf("Container.destroy.failed: %v %s", c, bytes)
		return
	}

	_, err := c.client.Destroy(c.handle)
	if err != nil {
		utils.Logger("Container").Warnf("Error destroying container: %s", err.Error())
	}
	c.handle = ""
}

func (c *Container) CloseAllConnections() {
	c.client.Disconnect()
}

func (c *Container) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {
	panic("Regenerate protobufs for discard output")

	resourceLimits := warden.ResourceLimits{Nproc: &nproc_limit,
		Nofile: &file_descriptor_limit}
	req := warden.SpawnRequest{Handle: &c.handle, Script: &script,
		Rlimits: &resourceLimits}
	rsp, err := c.client.SpawnWithRequest(&req)
	if err != nil {
		utils.Logger("Container").Warnf("Error spawning container: %s", err.Error())
		return nil, err
	}
	return rsp, nil
}

func (c *Container) Link(jobId uint32) (*warden.LinkResponse, error) {
	rsp, err := c.client.Link(c.handle, jobId)
	if err != nil {
		utils.Logger("Container").Warnf("Error linking container: %s", err.Error())
		return nil, err
	}
	return rsp, nil
}

func (c *Container) CopyOut(sourcePath, destinationPath string, uid int) error {
	_, err := c.client.CopyOut(c.handle, sourcePath, destinationPath, fmt.Sprintf("%d", uid))
	if err != nil {
		utils.Logger("Container").Warnf("Error CopyOut container: %s", err.Error())
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

	c.NetworkPorts["host_port"] = netResponse.GetHostPort()
	c.NetworkPorts["container_port"] = netResponse.GetContainerPort()

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
