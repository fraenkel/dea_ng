package protocol

import (
	"dea/config"
	ds "dea/directory_server"
	"dea/starting"
	"time"
)

const VERSION = "0.0.1"

type HelloMessage struct {
	Id      string "id"
	Ip      string "ip"
	Port    uint16 "port"
	Version string "version"
}

func NewHelloMessage(uuid string, ip string, port uint16) *HelloMessage {
	return &HelloMessage{
		Id:      uuid,
		Ip:      ip,
		Port:    port,
		Version: VERSION,
	}
}

type GoodbyeMessage struct {
	Id           string         "id"
	Ip           string         "ip"
	AppIdToCount map[string]int "app_id_to_count"
	Version      string         "version"
}

func NewGoodbyeMessage(uuid string, ip string, appIdToCount map[string]int) *GoodbyeMessage {
	return &GoodbyeMessage{
		Id:           uuid,
		Ip:           ip,
		AppIdToCount: appIdToCount,
		Version:      VERSION,
	}
}

type HeartbeatInstanceV1 struct {
	CCPartition string    "cc_partition"
	Droplet     string    "droplet"
	Version     string    "version"
	Id          string    "instance"
	Index       int       "index"
	State       string    "state"
	Timestamp   time.Time "state_timestamp"
}

type HeartbeatResponseV1 struct {
	Heartbeats []HeartbeatInstanceV1 "droplets"
	Uuid       string                "dea"
}

func NewHeartbeatResponseV1(uuid string, instances []*starting.Instance) *HeartbeatResponseV1 {
	heartbeats := make([]HeartbeatInstanceV1, len(instances))
	for i, instance := range instances {
		heartbeats[i] = HeartbeatInstanceV1{
			CCPartition: instance.CCPartition(),
			Droplet:     instance.ApplicationId(),
			Version:     instance.ApplicationVersion(),
			Id:          instance.Id(),
			Index:       instance.Index(),
			State:       string(instance.State()),
			Timestamp:   instance.StateTimestamp(),
		}
	}
	response := HeartbeatResponseV1{
		Heartbeats: heartbeats,
		Uuid:       uuid,
	}

	return &response
}

type ExitMessage struct {
	CCPartition     string     "cc_partition"
	Droplet         string     "droplet"
	Version         string     "version"
	Id              string     "instance"
	Index           int        "index"
	Reason          string     "reason"
	ExitStatus      int32      "exit_status"
	ExitDescription string     "exit_description"
	CrashTimestamp  *time.Time "crash_timestamp"
}

func NewExitMessage(i starting.Instance, reason string) *ExitMessage {
	exitm := ExitMessage{
		CCPartition:     i.CCPartition(),
		Droplet:         i.ApplicationId(),
		Version:         i.ApplicationVersion(),
		Id:              i.Id(),
		Index:           i.Index(),
		Reason:          reason,
		ExitStatus:      i.ExitStatus(),
		ExitDescription: i.ExitDescription(),
	}

	if i.State() == starting.STATE_CRASHED {
		sTime := i.StateTimestamp()
		exitm.CrashTimestamp = &sTime
	}

	return &exitm
}

type AdvertiseMessage struct {
	id              string         "id"
	stacks          []string       "stacks"
	availableMemory float64        "available_memory"
	availableDisk   float64        "available_disk"
	appCounts       map[string]int "app_id_to_count"
}

func NewAdvertiseMessage(id string, stacks []string, availableMemory, availableDisk float64, appCounts map[string]int) *AdvertiseMessage {
	return &AdvertiseMessage{
		id:              id,
		stacks:          stacks,
		availableMemory: availableMemory,
		availableDisk:   availableDisk,
		appCounts:       appCounts,
	}
}

type DeaStatusResponse struct {
	HelloMessage
	MaxMemory      float64 "max_memory"
	ReservedMemory uint    "reserved_memory"
	UsedMemory     uint    "used_memory"
	NumClients     *uint32 "num_clients"
}

func NewDeaStatusResponse(uuid string, ip string, port uint16, memoryCapacity float64, reservedMemory, usedMemory uint) *DeaStatusResponse {
	response := DeaStatusResponse{
		MaxMemory:      memoryCapacity,
		ReservedMemory: reservedMemory,
		UsedMemory:     usedMemory,
	}
	response.HelloMessage = *NewHelloMessage(uuid, ip, port)

	return &response
}

type FindDropletResponse struct {
	Dea            string            `json:"dea"`
	Droplet        string            `json:"droplet"`
	Version        string            `json:"version"`
	Instance       string            `json:"instance"`
	Index          int               `json:"index"`
	State          string            `json:"state"`
	StateTimestamp time.Time         `json:"state_timestamp"`
	FileUri        string            `json:"file_uri"`
	FileUriV2      *string           `json:"file_uri_v2, omitempty"`
	Credentials    []string          `json:"credentials"`
	Staged         string            `json:"staged"`
	Stats          *FindDropletStats `json:"stats,omitempty"`
}

type FindDropletStats struct {
	Name        string           "name"
	Uris        []string         "uris"
	Host        string           "host"
	Port        uint32           "port"
	Uptime      float64          "uptime"
	MemoryQuota config.Memory    "mem_quota"
	DiskQuota   config.Disk      "disk_quota"
	FdsQuota    uint64           "fds_quota"
	Usage       FindDropletUsage `json:"usage"`
}

type FindDropletUsage struct {
	Time     string
	CPU      float32     "cpu"
	MemoryKB uint64      "mem"
	Disk     config.Disk "disk"
}

func NewFindDropletResponse(uuid string, localIp string, i *starting.Instance,
	dirServerV1 *ds.DirectoryServerV1, dirServerV2 *ds.DirectoryServerV2,
	request map[string]interface{}) *FindDropletResponse {
	rsp := FindDropletResponse{
		Dea:            uuid,
		Droplet:        i.ApplicationId(),
		Version:        i.ApplicationVersion(),
		Instance:       i.Id(),
		Index:          i.Index(),
		State:          string(i.State()),
		StateTimestamp: i.StateTime(i.State()),
		FileUri:        dirServerV1.Uri,
		Credentials:    dirServerV1.Credentials,
		Staged:         "/" + i.Id(),
	}

	if path, exists := request["path"]; exists {
		fileUriV2 := dirServerV2.Instance_file_url_for(i.Id(), path.(string))
		rsp.FileUriV2 = &fileUriV2
	}

	if _, exists := request["include_stats"]; exists && i.State() == starting.STATE_RUNNING {
		rspStats := FindDropletStats{
			Name:        i.ApplicationName(),
			Uris:        i.ApplicationUris(),
			Host:        localIp,
			Port:        i.HostPort(),
			Uptime:      time.Now().Sub(i.StateTime(starting.STATE_STARTING)).Seconds(),
			MemoryQuota: i.MemoryLimit(),
			DiskQuota:   i.DiskLimit(),
			FdsQuota:    i.FileDescriptorLimit(),
			Usage: FindDropletUsage{
				Time:     time.Now().Format(time.RFC3339),
				CPU:      i.Computed_pcpu(),
				MemoryKB: uint64(i.UsedMemory() / config.Kibi),
				Disk:     i.UsedDisk(),
			},
		}

		// Purposefully omitted, as I'm not sure what purpose it serves.
		// cores
		rsp.Stats = &rspStats
	}

	return &rsp
}
