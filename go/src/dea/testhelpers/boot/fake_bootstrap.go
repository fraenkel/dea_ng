package boot

import (
	dboot "dea/boot"
	cfg "dea/config"
	"dea/droplet"
	rm "dea/resource_manager"
	rtr "dea/router_client"
	"dea/staging"
	"dea/starting"
	"github.com/cloudfoundry/yagnats"
)

type FakeBootstrap struct {
	Cfg             cfg.Config
	IManager        dboot.InstanceManager
	ResMgr          rm.ResourceManager
	IRegistry       starting.InstanceRegistry
	DropRegistry    droplet.DropletRegistry
	RtrClient       rtr.RouterClient
	Localip         string
	Snap            dboot.Snapshot
	NatsClient      yagnats.NATSClient
	StagingRegistry *staging.StagingTaskRegistry

	SendExitMessageInstance *starting.Instance
	SendExitMessageReason   string

	SendInstanceStopMessageInstance *starting.Instance

	SendHeartbeatInvoked bool
}

func (fb *FakeBootstrap) Config() *cfg.Config {
	return &fb.Cfg
}

func (fb *FakeBootstrap) InstanceManager() dboot.InstanceManager {
	return fb.IManager
}
func (fb *FakeBootstrap) StagingTaskRegistry() *staging.StagingTaskRegistry {
	return fb.StagingRegistry
}
func (fb *FakeBootstrap) Nats() yagnats.NATSClient {
	return fb.NatsClient
}

func (fb *FakeBootstrap) ResourceManager() rm.ResourceManager {
	return fb.ResMgr
}
func (fb *FakeBootstrap) InstanceRegistry() starting.InstanceRegistry {
	return fb.IRegistry
}
func (fb *FakeBootstrap) DropletRegistry() droplet.DropletRegistry {
	return fb.DropRegistry
}
func (fb *FakeBootstrap) RouterClient() rtr.RouterClient {
	return fb.RtrClient
}
func (fb *FakeBootstrap) LocalIp() string {
	return fb.Localip
}
func (fb *FakeBootstrap) SendExitMessage(i *starting.Instance, reason string) {
	fb.SendExitMessageInstance = i
	fb.SendExitMessageReason = reason
}
func (fb *FakeBootstrap) SendInstanceStopMessage(i *starting.Instance) {
	fb.SendInstanceStopMessageInstance = i
}
func (fb *FakeBootstrap) SendHeartbeat() {
	fb.SendHeartbeatInvoked = true
}
func (fb *FakeBootstrap) Snapshot() dboot.Snapshot {
	return fb.Snap
}
