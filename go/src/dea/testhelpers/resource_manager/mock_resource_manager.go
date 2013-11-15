package resource_manager

type MockResourceManager struct {
	Memory    float64
	Disk      float64
	AppCounts map[string]int
}

func (rm MockResourceManager) MemoryCapacity() float64 {
	return 1.0
}
func (rm MockResourceManager) DiskCapacity() float64 {
	return 1.0
}
func (rm MockResourceManager) AppIdToCount() map[string]int {
	return rm.AppCounts
}
func (rm MockResourceManager) RemainingMemory() float64 {
	return rm.Memory
}
func (rm MockResourceManager) ReservedMemory() float64 {
	return 1.0
}
func (rm MockResourceManager) UsedMemory() float64 {
	return 1.0
}
func (rm MockResourceManager) CanReserve(memory, disk float64) bool {
	return true
}
func (rm MockResourceManager) RemainingDisk() float64 {
	return rm.Disk
}
func (rm MockResourceManager) NumberReservable(memory, disk uint64) uint {
	return 1
}
func (rm MockResourceManager) AvailableMemoryRatio() float64 {
	return 1.0
}
func (rm MockResourceManager) AvailableDiskRatio() float64 {
	return 1.0
}
