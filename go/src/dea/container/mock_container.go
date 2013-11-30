package container

import "github.com/cloudfoundry/gordon"

type MockContainer struct {
	MHandle      string
	MPath        string
	MHostIp      string
	MNetworkPort map[string]uint32

	MRunScript       string
	MRunScriptFunc   func()
	MRunScriptStdout string
	MRunScriptStderr string
	MRunScriptError  error

	MInfoError    error
	MInfoResponse *warden.InfoResponse

	MCopyOutSrc      string
	MCopyOutDest     string
	MCopyOutCount    int
	MCopyOutCallback func(dest string)

	MStopHandle string
	MStopError  error

	MUpdatePathAndIpPath   string
	MUpdatePathAndIpHostIp string
	MUpdatePathAndIpError  error

	MCreateHandle      string
	MCreateError       error
	MCreateBindMounts  []*warden.CreateRequest_BindMount
	MCreateDiskLimit   uint64
	MCreateMemoryLimit uint64
	MCreateNetwork     bool

	MSpawnError               error
	MSpawnScript              string
	MSpawnJobId               uint32
	MSpawnFileDescriptorLimit uint64
	MSpawnNProc               uint64

	MLinkJobId    uint32
	MLinkResponse *warden.LinkResponse
	MLinkError    error

	MDestroyInvoked bool

	MCloseAllConnectionsInvoked bool
}

func (m *MockContainer) Setup(handle string, hostPort, containerPort uint32) {
	m.MHandle = handle
	if m.MNetworkPort == nil {
		m.MNetworkPort = make(map[string]uint32)
	}
	m.MNetworkPort[HOST_PORT] = hostPort
	m.MNetworkPort[CONTAINER_PORT] = containerPort
}

func (m *MockContainer) Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error {
	m.MCreateBindMounts = bind_mounts
	m.MCreateDiskLimit = disk_limit
	m.MCreateMemoryLimit = memory_limit
	m.MCreateNetwork = network
	if m.MCreateError == nil {
		m.MHandle = m.MCreateHandle
	}
	return m.MCreateError
}

func (m *MockContainer) Stop() error {
	m.MStopHandle = m.MHandle
	return m.MStopError
}
func (m *MockContainer) CloseAllConnections() {
	m.MCloseAllConnectionsInvoked = true
}
func (m *MockContainer) Destroy() {
	m.MDestroyInvoked = true
}
func (m *MockContainer) RunScript(script string) (*warden.RunResponse, error) {
	m.MRunScript = script
	if m.MRunScriptFunc != nil {
		m.MRunScriptFunc()
	}
	runRsp := &warden.RunResponse{
		Stdout: &m.MRunScriptStdout,
		Stderr: &m.MRunScriptStderr,
	}
	return runRsp, m.MRunScriptError
}
func (m *MockContainer) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {
	if m.MSpawnError != nil {
		return nil, m.MSpawnError
	}

	m.MSpawnScript = script
	m.MSpawnFileDescriptorLimit = file_descriptor_limit
	m.MSpawnNProc = nproc_limit
	rsp := warden.SpawnResponse{JobId: &m.MSpawnJobId}
	return &rsp, nil
}
func (m *MockContainer) Link(jobId uint32) (*warden.LinkResponse, error) {
	m.MLinkJobId = jobId
	return m.MLinkResponse, m.MLinkError
}
func (m *MockContainer) CopyOut(sourcePath, destinationPath string, uid int) error {
	m.MCopyOutSrc = sourcePath
	m.MCopyOutDest = destinationPath
	m.MCopyOutCount++
	if m.MCopyOutCallback != nil {
		m.MCopyOutCallback(destinationPath)
	}
	
	return nil
}
func (m *MockContainer) Info() (*warden.InfoResponse, error) {
	return m.MInfoResponse, m.MInfoError
}
func (m *MockContainer) Update_path_and_ip() error {
	m.MPath = m.MUpdatePathAndIpPath
	m.MHostIp = m.MUpdatePathAndIpHostIp
	return m.MUpdatePathAndIpError
}
func (m *MockContainer) Handle() string {
	return m.MHandle
}
func (m *MockContainer) Path() string {
	return m.MPath
}
func (m *MockContainer) HostIp() string {
	return m.MHostIp
}
func (m *MockContainer) NetworkPort(port string) uint32 {
	return m.MNetworkPort[port]
}
