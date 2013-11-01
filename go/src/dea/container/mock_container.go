package container

import "github.com/cloudfoundry/gordon"

type MockContainer struct {
	MHandle     string
	MPath       string
	MHostIp     string
	MNetworkPort map[string]uint32
}

func (m MockContainer) Setup(handle string, hostPort, containerPort uint32) {
}

func (m MockContainer) Create(bind_mounts []*warden.CreateRequest_BindMount, disk_limit uint64, memory_limit uint64, network bool) error {
	return nil
}

func (m MockContainer) Stop() error {
	return nil
}
func (m MockContainer) CloseAllConnections() {}
func (m MockContainer) Destroy()             {}
func (m MockContainer) RunScript(script string) (*warden.RunResponse, error) {
	return nil, nil
}
func (m MockContainer) Spawn(script string, file_descriptor_limit, nproc_limit uint64, discard_output bool) (*warden.SpawnResponse, error) {
	return nil, nil
}
func (m MockContainer) Link(jobId uint32) (*warden.LinkResponse, error) {
	return nil, nil
}
func (m MockContainer) CopyOut(sourcePath, destinationPath string, uid int) error {
	return nil
}
func (m MockContainer) Info() (*warden.InfoResponse, error) {
	return nil, nil
}
func (m MockContainer) Update_path_and_ip() error {
	return nil
}
func (m MockContainer) Handle() string {
	return m.MHandle
}
func (m MockContainer) Path() string {
	return m.MPath
}
func (m MockContainer) HostIp() string {
	return m.MHostIp
}
func (m MockContainer) NetworkPort(port string) uint32 {
	return m.MNetworkPort[port]
}
