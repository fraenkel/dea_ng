package dea

import (
	"dea/config"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	"net/url"
	"time"
)

type Bootstrap interface {
	Config() *config.Config
	InstanceManager() InstanceManager
	Snapshot() Snapshot
	ResourceManager() ResourceManager
	InstanceRegistry() InstanceRegistry
	StagingTaskRegistry() StagingTaskRegistry
	DropletRegistry() DropletRegistry
	RouterClient() RouterClient
	Nats() yagnats.NATSClient
	LocalIp() string
	SendHeartbeat()
}

type CPUStat struct {
	Timestamp time.Time
	Usage     uint64
}

type Stats struct {
	UsedMemory   config.Memory
	UsedDisk     config.Disk
	ComputedPCPU float32
	CpuSamples   []CPUStat
}

type StagingTaskUrlMaker interface {
	UrlForStagingTask(task_id, file_path string) string
}

type StagingTaskRegistry interface {
	NewStagingTask(staging_message StagingMessage, logger *steno.Logger) StagingTask
	Register(task StagingTask)
	Unregister(task StagingTask)

	Task(task_id string) StagingTask
	Tasks() []StagingTask

	ReservedMemory() config.Memory
	ReservedDisk() config.Disk
}

type StagingBuildpack struct {
	Url *url.URL
	Key string
}

type StagingMessage interface {
	StartMessage() StartMessage
	App_id() string
	Task_id() string
	AdminBuildpacks() []StagingBuildpack
	Properties() map[string]interface{}

	Buildpack_cache_upload_uri() *url.URL
	Buildpack_cache_download_uri() *url.URL
	Upload_uri() *url.URL
	Download_uri() *url.URL

	AsMap() map[string]interface{}
}

type Callback func(e error) error

type StagingTask interface {
	Id() string
	StagingMessage() StagingMessage
	StagingConfig() config.StagingConfig
	MemoryLimit() config.Memory
	DiskLimit() config.Disk
	Start()
	Stop(callback Callback)
	StreamingLogUrl(maker StagingTaskUrlMaker) string
	DetectedBuildpack() string
	DropletSHA1() string
	Path_in_container(pathSuffix string) string
	SetAfter_setup_callback(callback Callback)
	SetAfter_complete_callback(callback Callback)
	SetAfter_stop_callback(callback Callback)
	StagingTimeout() time.Duration
}

type DropletRegistry interface {
	Get(sha1 string) Droplet
	Remove(sha1 string) Droplet
	Size() int
	SHA1s() []string
}

type Droplet interface {
	SHA1() string
	Dir() string
	Path() string
	Exists() bool
	Download(uri string) error
	Local_copy(source string) error
	Destroy()
}

type ResourceManager interface {
	MemoryCapacity() float64
	DiskCapacity() float64
	AppIdToCount() map[string]int
	RemainingMemory() float64
	ReservedMemory() float64
	UsedMemory() float64
	CanReserve(memory, disk float64) bool
	GetConstrainedResource(memory config.Memory, disk config.Disk) string
	RemainingDisk() float64
	NumberReservable(memory, disk uint64) uint
	AvailableMemoryRatio() float64
	AvailableDiskRatio() float64
}

type Snapshot interface {
	Path() string
	Load() error
	Save()
}

type RouterClient interface {
	RegisterInstance(i Instance, opts map[string]interface{}) error
	UnregisterInstance(i Instance, opts map[string]interface{}) error
	Greet(callback yagnats.Callback) error
	Register_directory_server(host string, port uint32, uri string) error
	Unregister_directory_server(host string, port uint32, uri string) error
}

type StatCollector interface {
	Start() bool
	Stop()
	GetStats() Stats
	Retrieve_stats(now time.Time)
}

type EnvStrategy interface {
	ExportedSystemEnvironmentVariables() [][]string
	VcapApplication() map[string]interface{}
	StartMessage() StartMessage
}

type Responder interface {
	Start()
	Stop()
}

type LocatorResponder interface {
	Advertise()
}

type DirectoryServerV2 interface {
	Stop() error
}

type Task interface {
	Stop(callback Callback)
}
