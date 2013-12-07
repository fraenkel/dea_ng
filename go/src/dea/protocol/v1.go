package protocol

import (
	"dea/config"
	ds "dea/directory_server"
	"dea/starting"
	"time"
)

const VERSION = "0.0.1"

type HelloMessage struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	Port    uint16 `json:"port"`
	Version string `json:"version"`
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
	Id           string         `json:"id"`
	Ip           string         `json:"ip"`
	AppIdToCount map[string]int `json:"app_id_to_count"`
	Version      string         `json:"version"`
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
	CCPartition string    `json:"cc_partition"`
	Droplet     string    `json:"droplet"`
	Version     string    `json:"version"`
	Id          string    `json:"instance"`
	Index       int       `json:"index"`
	State       string    `json:"state"`
	Timestamp   time.Time `json:"state_timestamp"`
}

type HeartbeatResponseV1 struct {
	Heartbeats []HeartbeatInstanceV1 `json:"droplets"`
	Uuid       string                `json:"dea"`
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
	CCPartition     string     `json:"cc_partition"`
	Droplet         string     `json:"droplet"`
	Version         string     `json:"version"`
	Id              string     `json:"instance"`
	Index           int        `json:"index"`
	Reason          string     `json:"reason"`
	ExitStatus      int64      `json:"exit_status"`
	ExitDescription string     `json:"exit_description"`
	CrashTimestamp  *time.Time `json:"crash_timestamp"`
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
	ID                  string                 `json:"id"`
	Stacks              []string               `json:"stacks"`
	AvailableMemory     float64                `json:"available_memory"`
	AvailableDisk       float64                `json:"available_disk"`
	AppCounts           map[string]int         `json:"app_id_to_count"`
	PlacementProperties map[string]interface{} `json:"placement_properties"`
}

func NewAdvertiseMessage(id string, stacks []string, availableMemory, availableDisk float64, appCounts map[string]int, placementProperties map[string]interface{}) *AdvertiseMessage {
	return &AdvertiseMessage{
		ID:                  id,
		Stacks:              stacks,
		AvailableMemory:     availableMemory,
		AvailableDisk:       availableDisk,
		AppCounts:           appCounts,
		PlacementProperties: placementProperties,
	}
}

type DeaStatusResponse struct {
	Id             string  `json:"id"`
	Ip             string  `json:"ip"`
	Port           uint16  `json:"port"`
	Version        string  `json:"version"`
	MaxMemory      float64 `json:"max_memory"`
	ReservedMemory uint    `json:"reserved_memory"`
	UsedMemory     uint    `json:"used_memory"`
	NumClients     *uint32 `json:"num_clients"`
}

func NewDeaStatusResponse(uuid string, ip string, port uint16, memoryCapacity float64, reservedMemory, usedMemory uint) *DeaStatusResponse {
	response := DeaStatusResponse{
		Id:             uuid,
		Ip:             ip,
		Port:           port,
		Version:        VERSION,
		MaxMemory:      memoryCapacity,
		ReservedMemory: reservedMemory,
		UsedMemory:     usedMemory,
	}

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
	Name        string           `json:"name"`
	Uris        []string         `json:"uris"`
	Host        string           `json:"host"`
	Port        uint32           `json:"port"`
	Uptime      float64          `json:"uptime"`
	MemoryQuota config.Memory    `json:"mem_quota"`
	DiskQuota   config.Disk      `json:"disk_quota"`
	FdsQuota    uint64           `json:"fds_quota"`
	Usage       FindDropletUsage `json:"usage"`
}

type FindDropletUsage struct {
	Time     string      `json:"time"`
	CPU      float32     `json:"cpu"`
	MemoryKB uint64      `json:"mem"`
	Disk     config.Disk `json:"disk"`
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
		stats := i.GetStats()
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
				CPU:      stats.ComputedPCPU,
				MemoryKB: uint64(stats.UsedMemory / config.Kibi),
				Disk:     stats.UsedDisk,
			},
		}

		// Purposefully omitted, as I'm not sure what purpose it serves.
		// cores
		rsp.Stats = &rspStats
	}

	return &rsp
}
