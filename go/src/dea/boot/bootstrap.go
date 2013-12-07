package boot

import (
	"dea/config"
	"dea/droplet"
	rm "dea/resource_manager"
	rtr "dea/router_client"
	"dea/staging"
	"dea/starting"
	"github.com/cloudfoundry/yagnats"
)

type Bootstrap interface {
	Config() *config.Config
	InstanceManager() InstanceManager
	Snapshot() Snapshot
	ResourceManager() rm.ResourceManager
	InstanceRegistry() starting.InstanceRegistry
	StagingTaskRegistry() *staging.StagingTaskRegistry
	DropletRegistry() droplet.DropletRegistry
	RouterClient() rtr.RouterClient
	Nats() yagnats.NATSClient
	LocalIp() string
	SendExitMessage(i *starting.Instance, reason string)
	SendInstanceStopMessage(instance *starting.Instance)
	SendHeartbeat()
}
