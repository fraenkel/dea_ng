package container

import (
	"dea"
	"github.com/cloudfoundry/gordon"
)

type FakeContainer struct {
	FHandle      string
	FPath        string
	FHostIp      string
	FNetworkPort map[string]uint32

	FRunScript       string
	FRunScriptFunc   func()
	FRunScriptStdout string
	FRunScriptStderr string
	FRunScriptError  error

	FInfoError    error
	FInfoResponse *warden.InfoResponse

	FCopyOutSrc      string
	FCopyOutDest     string
	FCopyOutCount    int
	FCopyOutCallback func(dest string)

	FStopHandle string
	FStopError  error

	FUpdatePathAndIpPath   string
	FUpdatePathAndIpHostIp string
	FUpdatePathAndIpError  error

	FCreateHandle      string
	FCreateError       error
	FCreateBindMounts  []*warden.CreateRequest_BindMount
	FCreateDiskLimit   uint64
	FCreateMemoryLimit uint64
	FCreateNetwork     bool

	FSpawnError               error
	FSpawnScript              string
	FSpawnJobId               uint32
	FSpawnFileDescriptorLimit uint64
	FSpawnNProc               uint64

	FLinkJobId    uint32
	FLinkResponse *warden.LinkResponse
	FLinkError    error

	FDestroyInvoked bool

	FCloseAllConnectionsInvoked bool
}

func (m *FakeContainer) Setup(handle string, hostPort, containerPort uint32) {
	m.FHandle = handle
	if m.FNetworkPort == nil {
		m.FNetworkPort = make(map[string]uint32)
	}
	m.FNetworkPort[dea.HOST_PORT] = hostPort
	m.FNetworkPort[dea.CONTAINER_PORT] = containerPort
}

func (m *FakeContainer) Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error {
	m.FCreateBindMounts = bind_mounts
	m.FCreateDiskLimit = disk_limit
	m.FCreateMemoryLimit = memory_limit
	m.FCreateNetwork = network
	if m.FCreateError == nil {
		m.FHandle = m.FCreateHandle
	}
	return m.FCreateError
}

func (m *FakeContainer) Stop() error {
	m.FStopHandle = m.FHandle
	return m.FStopError
}
func (m *FakeContainer) CloseAllConnections() {
	m.FCloseAllConnectionsInvoked = true
}
func (m *FakeContainer) Destroy() error {
	m.FDestroyInvoked = true
	return nil
}
func (m *FakeContainer) RunScript(script string) (*warden.RunResponse, error) {
	m.FRunScript = script
	if m.FRunScriptFunc != nil {
		m.FRunScriptFunc()
	}
	runRsp := &warden.RunResponse{
		Stdout: &m.FRunScriptStdout,
		Stderr: &m.FRunScriptStderr,
	}
	return runRsp, m.FRunScriptError
}
func (m *FakeContainer) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {
	if m.FSpawnError != nil {
		return nil, m.FSpawnError
	}

	m.FSpawnScript = script
	m.FSpawnFileDescriptorLimit = file_descriptor_limit
	m.FSpawnNProc = nproc_limit
	rsp := warden.SpawnResponse{JobId: &m.FSpawnJobId}
	return &rsp, nil
}
func (m *FakeContainer) Link(jobId uint32) (*warden.LinkResponse, error) {
	m.FLinkJobId = jobId
	return m.FLinkResponse, m.FLinkError
}
func (m *FakeContainer) CopyOut(sourcePath, destinationPath string, uid int) error {
	m.FCopyOutSrc = sourcePath
	m.FCopyOutDest = destinationPath
	m.FCopyOutCount++
	if m.FCopyOutCallback != nil {
		m.FCopyOutCallback(destinationPath)
	}

	return nil
}
func (m *FakeContainer) Info() (*warden.InfoResponse, error) {
	return m.FInfoResponse, m.FInfoError
}
func (m *FakeContainer) Update_path_and_ip() error {
	m.FPath = m.FUpdatePathAndIpPath
	m.FHostIp = m.FUpdatePathAndIpHostIp
	return m.FUpdatePathAndIpError
}
func (m *FakeContainer) Handle() string {
	return m.FHandle
}
func (m *FakeContainer) Path() string {
	return m.FPath
}
func (m *FakeContainer) HostIp() string {
	return m.FHostIp
}
func (m *FakeContainer) NetworkPort(port string) uint32 {
	return m.FNetworkPort[port]
}
