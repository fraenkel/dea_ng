package boot

import (
	"dea"
	cfg "dea/config"
	"github.com/cloudfoundry/yagnats"
)

type FakeBootstrap struct {
	Cfg             cfg.Config
	IManager        dea.InstanceManager
	ResMgr          dea.ResourceManager
	IRegistry       dea.InstanceRegistry
	DropRegistry    dea.DropletRegistry
	RtrClient       dea.RouterClient
	Localip         string
	Snap            dea.Snapshot
	NatsClient      yagnats.NATSClient
	StagingRegistry dea.StagingTaskRegistry

	SendExitMessageInstance dea.Instance
	SendExitMessageReason   string

	SendInstanceStopMessageInstance dea.Instance

	SendHeartbeatInvoked bool
}

func (fb *FakeBootstrap) Config() *cfg.Config {
	return &fb.Cfg
}

func (fb *FakeBootstrap) InstanceManager() dea.InstanceManager {
	return fb.IManager
}
func (fb *FakeBootstrap) StagingTaskRegistry() dea.StagingTaskRegistry {
	return fb.StagingRegistry
}
func (fb *FakeBootstrap) Nats() yagnats.NATSClient {
	return fb.NatsClient
}

func (fb *FakeBootstrap) ResourceManager() dea.ResourceManager {
	return fb.ResMgr
}
func (fb *FakeBootstrap) InstanceRegistry() dea.InstanceRegistry {
	return fb.IRegistry
}
func (fb *FakeBootstrap) DropletRegistry() dea.DropletRegistry {
	return fb.DropRegistry
}
func (fb *FakeBootstrap) RouterClient() dea.RouterClient {
	return fb.RtrClient
}
func (fb *FakeBootstrap) LocalIp() string {
	return fb.Localip
}
func (fb *FakeBootstrap) SendExitMessage(i dea.Instance, reason string) {
	fb.SendExitMessageInstance = i
	fb.SendExitMessageReason = reason
}
func (fb *FakeBootstrap) SendInstanceStopMessage(i dea.Instance) {
	fb.SendInstanceStopMessageInstance = i
}
func (fb *FakeBootstrap) SendHeartbeat() {
	fb.SendHeartbeatInvoked = true
}
func (fb *FakeBootstrap) Snapshot() dea.Snapshot {
	return fb.Snap
}
