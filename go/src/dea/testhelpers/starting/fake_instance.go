package starting

import (
	"dea"
	"dea/config"
	"time"
)

type FakeInstance struct {
	CurrentState dea.State

	Stopped   bool
	StopError error
}

func (fi *FakeInstance) StartMessage() dea.StartMessage {
	return nil
}
func (fi *FakeInstance) Stop(callback dea.Callback) {
	fi.Stopped = true
	if callback != nil {
		go callback(fi.StopError)
	}
}

func (fi *FakeInstance) CCPartition() string {
	return ""
}
func (fi *FakeInstance) Id() string {
	return ""
}
func (fi *FakeInstance) Index() int {
	return 0
}
func (fi *FakeInstance) ApplicationId() string {
	return ""
}
func (fi *FakeInstance) ApplicationName() string {
	return ""
}
func (fi *FakeInstance) ApplicationUris() []string {
	return nil
}
func (fi *FakeInstance) SetApplicationUris(uris []string) {
}
func (fi *FakeInstance) ApplicationVersion() string {
	return ""
}
func (fi *FakeInstance) SetApplicationVersion(version string) {
}
func (fi *FakeInstance) State() dea.State {
	return fi.CurrentState
}
func (fi *FakeInstance) SetState(s dea.State) {
	fi.CurrentState = s
}
func (fi *FakeInstance) StateTime(state dea.State) time.Time {
	return time.Now()
}
func (fi *FakeInstance) StateTimestamp() time.Time {
	return time.Now()
}
func (fi *FakeInstance) DropletSHA1() string {
	return ""
}

func (fi *FakeInstance) MemoryLimit() config.Memory {
	return 0
}
func (fi *FakeInstance) DiskLimit() config.Disk {
	return 0
}
func (fi *FakeInstance) FileDescriptorLimit() uint64 {
	return 0
}

func (fi *FakeInstance) GetStats() dea.Stats {
	return dea.Stats{}
}

func (fi *FakeInstance) Path() (string, error) {
	return "", nil
}

func (fi *FakeInstance) ExitStatus() int64 {
	return 0
}
func (fi *FakeInstance) ExitDescription() string {
	return ""
}

func (fi *FakeInstance) HostPort() uint32 {
	return 0
}
func (fi *FakeInstance) ContainerPort() uint32 {
	return 0
}

func (fi *FakeInstance) Start(callback func(error) error) {
}

func (fi *FakeInstance) SetId() {
}

func (fi *FakeInstance) IsPathAvailable() bool {
	return true
}
func (fi *FakeInstance) IsConsumingMemory() bool {
	return true
}
func (fi *FakeInstance) IsConsumingDisk() bool {
	return true
}

func (fi *FakeInstance) Attributes_and_stats() map[string]interface{} {
	return nil
}
func (fi *FakeInstance) Snapshot_attributes() map[string]interface{} {
	return nil
}
func (fi *FakeInstance) PrivateInstanceId() string {
	return ""
}
