package starting

import (
	"dea"
	cfg "dea/config"
)

func NewFakeInstanceRegistry() *FakeInstanceRegistry {
	return &FakeInstanceRegistry{
		instances: make([]dea.Instance, 0, 1),
	}
}

type FakeInstanceRegistry struct {
	instances []dea.Instance
}

func (fir *FakeInstanceRegistry) Instances() []dea.Instance {
	return fir.instances
}
func (fir *FakeInstanceRegistry) InstancesForApplication(app_id string) map[string]dea.Instance {
	return nil
}
func (fir *FakeInstanceRegistry) Register(instance dea.Instance) {
	fir.instances = append(fir.instances, instance)
}

func (fir *FakeInstanceRegistry) Unregister(instance dea.Instance) {
	is := fir.instances
	for i, v := range is {
		if v == instance {
			fir.instances = append(is[:i], is[i+1:]...)
		}
	}
}

func (fir *FakeInstanceRegistry) ChangeInstanceId(instance dea.Instance) {
}
func (fir *FakeInstanceRegistry) LookupInstance(instanceId string) dea.Instance {
	return nil
}
func (fir *FakeInstanceRegistry) StartReapers() {
}
func (fir *FakeInstanceRegistry) AppIdToCount() map[string]int {
	return nil
}
func (fir *FakeInstanceRegistry) ReservedMemory() cfg.Memory {
	return 0
}
func (fir *FakeInstanceRegistry) UsedMemory() cfg.Memory {
	return 0
}
func (fir *FakeInstanceRegistry) ReservedDisk() cfg.Disk {
	return 0
}
func (fir *FakeInstanceRegistry) ToHash() map[string]map[string]interface{} {
	return nil
}
func (fir *FakeInstanceRegistry) Instances_filtered_by_message(data map[string]interface{}, f func(dea.Instance)) {
}
