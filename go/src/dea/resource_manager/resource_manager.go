package resource_manager

import (
	"dea/config"
	"dea/staging"
	"dea/starting"
	"math"
)

type ResourceManager struct {
	memoryCapacityMB    float64
	diskCapacityMB      float64
	instanceRegistry    *starting.InstanceRegistry
	stagingTaskRegistry *staging.StagingTaskRegistry
}

var defaultConfig = &config.ResourcesConfig{
	MemoryMB:               8 * 1024,
	MemoryOvercommitFactor: 1,
	DiskMB:                 16 * 1024 * 1024,
	DiskOvercommitFactor:   1,
}

func getUint64(configVal, defaultVal uint64) uint64 {
	if configVal != 0 {
		return configVal
	}
	return defaultVal
}

func getFloat64(configVal, defaultVal float64) float64 {
	if configVal != 0 {
		return configVal
	}
	return defaultVal
}

func NewResourceManager(iRegistry *starting.InstanceRegistry,
	stRegistry *staging.StagingTaskRegistry,
	config *config.ResourcesConfig) *ResourceManager {

	memoryMb := getUint64(config.MemoryMB, defaultConfig.MemoryMB)
	memoryOvercommit := getFloat64(config.MemoryOvercommitFactor, defaultConfig.MemoryOvercommitFactor)
	memory := float64(memoryMb) * memoryOvercommit
	diskMb := getUint64(config.DiskMB, defaultConfig.DiskMB)
	diskOvercommit := getFloat64(config.DiskOvercommitFactor, defaultConfig.DiskOvercommitFactor)
	disk := float64(diskMb) * diskOvercommit

	return &ResourceManager{
		memoryCapacityMB:    memory,
		diskCapacityMB:      disk,
		instanceRegistry:    iRegistry,
		stagingTaskRegistry: stRegistry,
	}
}

func (rm *ResourceManager) MemoryCapacity() float64 {
	return rm.memoryCapacityMB
}

func (rm *ResourceManager) DiskCapacity() float64 {
	return rm.diskCapacityMB
}

func (rm *ResourceManager) AppIdToCount() map[string]int {
	return rm.instanceRegistry.AppIdToCount()
}

func (rm *ResourceManager) RemainingMemory() float64 {
	return rm.memoryCapacityMB - rm.ReservedMemory()
}

func (rm *ResourceManager) ReservedMemory() float64 {
	return float64((rm.instanceRegistry.ReservedMemory() +
		rm.stagingTaskRegistry.ReservedMemory()) / config.Mebi)
}

func (rm *ResourceManager) UsedMemory() float64 {
	return float64(rm.instanceRegistry.UsedMemory() / config.Mebi)
}

func (rm *ResourceManager) CanReserve(memory, disk float64) bool {
	return rm.RemainingMemory() > memory &&
		rm.RemainingDisk() > disk
}

func (rm *ResourceManager) reserved_disk() float64 {
	return float64((rm.instanceRegistry.ReservedDisk() +
		rm.stagingTaskRegistry.ReservedDisk()) / config.MB)
}

func (rm *ResourceManager) RemainingDisk() float64 {
	return rm.DiskCapacity() - rm.reserved_disk()
}

func (rm *ResourceManager) NumberReservable(memory, disk uint64) uint {
	if memory == 0 || disk == 0 {
		return 0
	}

	return uint(math.Min(rm.RemainingMemory()/float64(memory), rm.RemainingDisk()/float64(disk)))
}

func (rm *ResourceManager) AvailableMemoryRatio() float64 {
	return 1.0 - (rm.ReservedMemory() / rm.MemoryCapacity())
}

func (rm *ResourceManager) AvailableDiskRatio() float64 {
	return 1.0 - (rm.reserved_disk() / rm.DiskCapacity())
}
