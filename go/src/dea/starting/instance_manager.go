package starting

import (
	"dea"
	"dea/protocol"
	"dea/utils"
	"encoding/json"
	steno "github.com/cloudfoundry/gosteno"
)

const (
	EXIT_REASON_CRASHED = "CRASHED"
)

type instanceManager struct {
	dea.Bootstrap
	logger *steno.Logger
}

func NewInstanceManager(b dea.Bootstrap) dea.InstanceManager {
	return &instanceManager{
		Bootstrap: b,
		logger:    utils.Logger("InstanceManager", nil),
	}
}

func (im *instanceManager) StartApp(data map[string]interface{}) {
	instance := im.CreateInstance(data)
	if instance != nil {
		instance.Start(nil)
	}
}

func (im *instanceManager) CreateInstance(attributes map[string]interface{}) dea.Instance {
	instance := NewInstance(attributes, im.Config(), im.DropletRegistry(), im.LocalIp())
	if instance == nil {
		return nil
	}

	err := instance.Validate()
	if err != nil {
		im.logger.Warnf("Error validating instance: %s", err.Error())
		return nil
	}

	instance.On(Transition{dea.STATE_BORN, dea.STATE_CRASHED}, func() {
		im.sendCrashedMessage(instance)
	})

	if constrained := im.ResourceManager().GetConstrainedResource(instance.MemoryLimit(), instance.DiskLimit()); constrained != "" {
		im.logger.Errord(map[string]interface{}{
			"app":                  instance.ApplicationId(),
			"instance":             instance.Index(),
			"constrained_resource": constrained,
		}, "instance.start.insufficient-resource")
		instance.SetExitStatus(-1, "Not enough "+constrained+" resource available.")
		instance.SetState(dea.STATE_CRASHED)
		return nil
	}

	instance.Setup()

	instance.On(Transition{dea.STATE_STARTING, dea.STATE_CRASHED}, func() {
		im.sendCrashedMessage(instance)
	})

	instance.On(Transition{dea.STATE_STARTING, dea.STATE_RUNNING}, func() {
		// Notify others immediately
		im.SendHeartbeat()

		// Register with router
		im.RouterClient().RegisterInstance(instance, nil)

		im.Snapshot().Save()
	})

	instance.On(Transition{dea.STATE_RUNNING, dea.STATE_CRASHED}, func() {
		im.RouterClient().UnregisterInstance(instance, nil)
		im.sendCrashedMessage(instance)
		im.Snapshot().Save()
	})

	instance.On(Transition{dea.STATE_RUNNING, dea.STATE_STOPPING}, func() {
		im.RouterClient().UnregisterInstance(instance, nil)
		im.Snapshot().Save()
	})

	instance.On(Transition{dea.STATE_EVACUATING, dea.STATE_STOPPING}, func() {
		im.RouterClient().UnregisterInstance(instance, nil)
		im.Snapshot().Save()
	})

	instance.On(Transition{dea.STATE_STOPPING, dea.STATE_STOPPED}, func() {
		im.InstanceRegistry().Unregister(instance)
		instance.Destroy(nil)
	})

	im.InstanceRegistry().Register(instance)
	return instance
}

func (im *instanceManager) sendCrashedMessage(i dea.Instance) {
	exitm := protocol.NewExitMessage(i, EXIT_REASON_CRASHED)
	bytes, err := json.Marshal(exitm)
	if err != nil {
		im.logger.Error(err.Error())
		return
	}
	im.Nats().Publish("droplet.exited", bytes)
}
