package boot

import (
	"dea/starting"
	"dea/utils"
	steno "github.com/cloudfoundry/gosteno"
)

const (
	EXIT_REASON_CRASHED = "CRASHED"
)

type InstanceManager interface {
	CreateInstance(attributes map[string]interface{}) *starting.Instance
	StartApp(attributes map[string]interface{})
}

type instanceManager struct {
	Bootstrap
	logger *steno.Logger
}

func NewInstanceManager(b Bootstrap) InstanceManager {
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

func (im *instanceManager) CreateInstance(attributes map[string]interface{}) *starting.Instance {
	instance := starting.NewInstance(attributes, im.Config(), im.DropletRegistry(), im.LocalIp())
	if instance == nil {
		return nil
	}

	err := instance.Validate()
	if err != nil {
		im.logger.Warnf("Error validating instance: %s", err.Error())
		return nil
	}

	instance.On(starting.Transition{starting.STATE_BORN, starting.STATE_CRASHED}, func() {
		im.SendExitMessage(instance, EXIT_REASON_CRASHED)
	})

	if constrained := im.ResourceManager().GetConstrainedResource(instance.MemoryLimit(), instance.DiskLimit()); constrained != "" {
		im.logger.Errord(map[string]interface{}{
			"app":                  instance.ApplicationId(),
			"instance":             instance.Index(),
			"constrained_resource": constrained,
		}, "instance.start.insufficient-resource")
		instance.SetExitStatus(-1, "Not enough "+constrained+" resource available.")
		instance.SetState(starting.STATE_CRASHED)
		return nil
	}

	instance.Setup()

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_CRASHED}, func() {
		im.SendExitMessage(instance, EXIT_REASON_CRASHED)
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_RUNNING}, func() {
		// Notify others immediately
		im.SendHeartbeat()

		// Register with router
		im.RouterClient().RegisterInstance(instance, nil)

		im.Snapshot().Save()
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_STOPPING}, func() {
		im.SendInstanceStopMessage(instance)
	})

	instance.On(starting.Transition{starting.STATE_RUNNING, starting.STATE_CRASHED}, func() {
		im.RouterClient().UnregisterInstance(instance, nil)
		im.SendExitMessage(instance, EXIT_REASON_CRASHED)
		im.Snapshot().Save()
	})

	instance.On(starting.Transition{starting.STATE_RUNNING, starting.STATE_STOPPING}, func() {
		im.RouterClient().UnregisterInstance(instance, nil)
		im.SendInstanceStopMessage(instance)
		im.Snapshot().Save()
	})

	instance.On(starting.Transition{starting.STATE_STOPPING, starting.STATE_STOPPED}, func() {
		im.InstanceRegistry().Unregister(instance)
		go instance.Destroy()
	})

	im.InstanceRegistry().Register(instance)
	return instance
}
