package dea

import (
	"dea/config"
	"time"
)

type State string

const (
	STATE_BORN       State = "born"
	STATE_STARTING   State = "starting"
	STATE_RUNNING    State = "running"
	STATE_STOPPING   State = "stopping"
	STATE_STOPPED    State = "stopped"
	STATE_CRASHED    State = "crashed"
	STATE_DELETED    State = "deleted"
	STATE_RESUMING   State = "resuming"
	STATE_EVACUATING State = "evacuating"
)

type StartMessage interface {
	Name() string
	Index() int
	Version() string
	Uris() []string
	Limits() map[string]uint64
	MemoryLimitMB() uint64
	DiskLimitMB() uint64
	Services() []map[string]interface{}
	Environment() map[string]string
}

type InstanceUrlMaker interface {
	UrlForInstance(instance_id, file_path string) string
}

type Instance interface {
	StartMessage() StartMessage
	Stop(callback Callback)

	CCPartition() string
	Id() string
	Index() int
	ApplicationId() string
	ApplicationName() string
	ApplicationUris() []string
	SetApplicationUris(uris []string)
	ApplicationVersion() string
	SetApplicationVersion(version string)
	State() State
	SetState(State)
	StateTime(state State) time.Time
	StateTimestamp() time.Time
	DropletSHA1() string

	MemoryLimit() config.Memory
	DiskLimit() config.Disk
	FileDescriptorLimit() uint64

	GetStats() Stats
	Path() (string, error)

	ExitStatus() int64
	ExitDescription() string

	HostPort() uint32
	ContainerPort() uint32

	Start(callback func(error) error)
	SetId()

	IsPathAvailable() bool
	IsConsumingMemory() bool
	IsConsumingDisk() bool

	Attributes_and_stats() map[string]interface{}
	Snapshot_attributes() map[string]interface{}
	PrivateInstanceId() string
}

type InstanceManager interface {
	CreateInstance(attributes map[string]interface{}) Instance
	StartApp(attributes map[string]interface{})
}

type InstanceRegistry interface {
	Instances() []Instance
	InstancesForApplication(app_id string) map[string]Instance
	Instances_filtered_by_message(data map[string]interface{}, f func(Instance))
	Register(instance Instance)
	Unregister(instance Instance)
	ChangeInstanceId(instance Instance)
	LookupInstance(instanceId string) Instance
	StartReapers()
	AppIdToCount() map[string]int
	ReservedMemory() config.Memory
	UsedMemory() config.Memory
	ReservedDisk() config.Disk
	ToHash() map[string]map[string]interface{}
}
