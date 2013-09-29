package dea

type ResourceManager struct {
	memoryCapacity      float32
	diskCapacity        float32
	instanceRegistry    *InstanceRegistry
	stagingTaskRegistry *StagingTaskRegistry
}

var defaultConfig = &ResourcesConfig{
	MemoryMb:               8 * 1024,
	MemoryOvercommitFactor: 1,
	DiskMb:                 16 * 1024 * 1024,
	DiskOvercommitFactor:   1,
}

func getInt(configVal, defaultVal int) int {
	if configVal != 0 {
		return configVal
	}
	return defaultVal
}

func getFloat32(configVal, defaultVal float32) float32 {
	if configVal != 0 {
		return configVal
	}
	return defaultVal
}

func NewResourceManager(iRegistry *InstanceRegistry,
	stRegistry *StagingTaskRegistry,
	config *ResourcesConfig) *ResourceManager {

	memoryMb := getInt(config.MemoryMb, defaultConfig.MemoryMb)
	memoryOvercommit := getFloat32(config.MemoryOvercommitFactor, defaultConfig.MemoryOvercommitFactor)
	memory := float32(memoryMb) * memoryOvercommit
	diskMb := getInt(config.DiskMb, defaultConfig.DiskMb)
	diskOvercommit := getFloat32(config.DiskOvercommitFactor, defaultConfig.DiskOvercommitFactor)
	disk := float32(diskMb) * diskOvercommit

	return &ResourceManager{
		memoryCapacity:      memory,
		diskCapacity:        disk,
		instanceRegistry:    iRegistry,
		stagingTaskRegistry: stRegistry,
	}
}

func (rm *ResourceManager) MemoryCapacity() float32 {
	return rm.memoryCapacity
}

func (rm *ResourceManager) DiskCapacity() float32 {
	return rm.diskCapacity
}

/*


    def app_id_to_count
      @instance_registry.app_id_to_count
    end

    def could_reserve?(memory, disk)
      (remaining_memory > memory) && (remaining_disk > disk)
    end

    def number_reservable(memory, disk)
      return 0 if memory.zero? || disk.zero?
      [remaining_memory / memory, remaining_disk / disk].min
    end

    def available_memory_ratio
      1.0 - (reserved_memory.to_f / memory_capacity)
    end

    def available_disk_ratio
      1.0 - (reserved_disk.to_f / disk_capacity)
    end

    def reserved_memory
      total_mb(@instance_registry, :reserved_memory_bytes) +
        total_mb(@staging_task_registry, :reserved_memory_bytes)
    end

    def used_memory
      total_mb(@instance_registry, :used_memory_bytes)
    end

    def reserved_disk
      total_mb(@instance_registry, :reserved_disk_bytes) +
        total_mb(@staging_task_registry, :reserved_disk_bytes)
    end

    def remaining_memory
      memory_capacity - reserved_memory
    end

    def remaining_disk
      disk_capacity - reserved_disk
    end

    private

    def total_mb(registry, resource_name)
      bytes_to_mb(registry.public_send(resource_name))
    end

    def bytes_to_mb(bytes)
      bytes / (1024 * 1024)
    end
  end
end
*/
