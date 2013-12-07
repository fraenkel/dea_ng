package resource_manager

import cfg "dea/config"

type FakeResourceManager struct {
	Memory    float64
	Disk      float64
	AppCounts map[string]int

	Reserve             bool
	ConstrainedResource string
	Reservable          uint
	MemoryRatio         float64
	DiskRatio           float64
}

func (rm FakeResourceManager) MemoryCapacity() float64 {
	return 1.0
}
func (rm FakeResourceManager) DiskCapacity() float64 {
	return 1.0
}
func (rm FakeResourceManager) AppIdToCount() map[string]int {
	return rm.AppCounts
}
func (rm FakeResourceManager) RemainingMemory() float64 {
	return rm.Memory
}
func (rm FakeResourceManager) ReservedMemory() float64 {
	return 1.0
}
func (rm FakeResourceManager) UsedMemory() float64 {
	return 1.0
}
func (rm FakeResourceManager) CanReserve(memory, disk float64) bool {
	return rm.Reserve
}
func (rm FakeResourceManager) GetConstrainedResource(memory cfg.Memory, disk cfg.Disk) string {
	return rm.ConstrainedResource
}

func (rm FakeResourceManager) RemainingDisk() float64 {
	return rm.Disk
}
func (rm FakeResourceManager) NumberReservable(memory, disk uint64) uint {
	return rm.Reservable
}
func (rm FakeResourceManager) AvailableMemoryRatio() float64 {
	return rm.MemoryRatio
}
func (rm FakeResourceManager) AvailableDiskRatio() float64 {
	return rm.DiskRatio
}
