package starting

import (
	cfg "dea/config"
	dstarting "dea/starting"
)

func NewFakeInstanceRegistry() dstarting.InstanceRegistry {
	return &FakeInstanceRegistry{
		instances: make([]*dstarting.Instance, 0, 1),
	}
}

type FakeInstanceRegistry struct {
	instances []*dstarting.Instance
}

func (fir *FakeInstanceRegistry) Instances() []*dstarting.Instance {
	return fir.instances
}
func (fir *FakeInstanceRegistry) InstancesForApplication(app_id string) map[string]*dstarting.Instance {
	return nil
}
func (fir *FakeInstanceRegistry) Register(instance *dstarting.Instance) {
	p := FakePromises{}
	instance.InstancePromises = &p
	instance.Task.TaskPromises = &p
	instance.StatCollector = &FakeStatCollector{}
	fir.instances = append(fir.instances, instance)
}

func (fir *FakeInstanceRegistry) Unregister(instance *dstarting.Instance) {
	is := fir.instances
	for i, v := range is {
		if v == instance {
			fir.instances = append(is[:i], is[i+1:]...)
		}
	}
}

func (fir *FakeInstanceRegistry) ChangeInstanceId(instance *dstarting.Instance) {
}
func (fir *FakeInstanceRegistry) LookupInstance(instanceId string) *dstarting.Instance {
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
func (fir *FakeInstanceRegistry) Instances_filtered_by_message(data map[string]interface{}, f func(*dstarting.Instance)) {
}
