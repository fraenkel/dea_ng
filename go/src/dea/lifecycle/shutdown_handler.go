package lifecycle

import (
	"dea"
	"dea/protocol"
	steno "github.com/cloudfoundry/gosteno"
	"os"
	"sync"
)

type ShutdownHandler interface {
	Shutdown(goodbyeMsg *protocol.GoodbyeMessage)
}

type shutdownHandler struct {
	shuttingDown        bool
	responders          []dea.Responder
	instanceRegistry    dea.InstanceRegistry
	stagingTaskRegistry dea.StagingTaskRegistry
	dropletRegistry     dea.DropletRegistry
	directoryServer     dea.DirectoryServerV2
	nats                *dea.Nats
	pidFile             *dea.PidFile
	logger              *steno.Logger
}

func NewShutdownHandler(responders []dea.Responder, iRegistry dea.InstanceRegistry, stagingRegistry dea.StagingTaskRegistry,
	dropletRegistry dea.DropletRegistry, dirServer dea.DirectoryServerV2, nats *dea.Nats,
	pidFile *dea.PidFile, logger *steno.Logger) ShutdownHandler {
	return &shutdownHandler{
		responders:          responders,
		instanceRegistry:    iRegistry,
		stagingTaskRegistry: stagingRegistry,
		dropletRegistry:     dropletRegistry,
		directoryServer:     dirServer,
		nats:                nats,
		pidFile:             pidFile,
		logger:              logger,
	}
}

func (s *shutdownHandler) Shutdown(goodbyeMsg *protocol.GoodbyeMessage) {
	if s.isShuttingDown() {
		return
	}

	s.shuttingDown = true

	s.removeDroplets()

	sendGoodbyeMessage(goodbyeMsg, s.nats.NatsClient, s.logger)
	stopResponders(s.responders)

	s.nats.Unsubscribe()

	s.directoryServer.Stop()

	s.stopTasks()
}

func (s *shutdownHandler) isShuttingDown() bool {
	return s.shuttingDown
}

func (s *shutdownHandler) removeDroplets() {
	all_shas := s.dropletRegistry.SHA1s()
	for _, sha := range all_shas {
		s.logger.Debugf("Removing droplet for sha=%s", sha)
		droplet := s.dropletRegistry.Remove(sha)
		droplet.Destroy()
	}
}

func (s *shutdownHandler) stopTasks() {
	pendingTasks := s.tasks()

	if len(pendingTasks) == 0 {
		s.terminate()
	}

	mutex := sync.Mutex{}
	mutex.Lock()

	for t, _ := range pendingTasks {
		myTask := t
		myTask.Stop(func(err error) error {
			empty := false

			mutex.Lock()
			delete(pendingTasks, myTask)
			empty = len(pendingTasks) == 0
			mutex.Unlock()

			if err != nil {
				s.logger.Warnf("%v failed to stop: %s", myTask, err.Error())
			} else {
				s.logger.Debugf("%v exited", myTask)
			}

			if empty {
				s.terminate()
			}

			return nil
		})
	}

	mutex.Unlock()
}

func (s *shutdownHandler) tasks() map[dea.Task]struct{} {
	all := make(map[dea.Task]struct{})

	for _, i := range s.instanceRegistry.Instances() {
		all[i] = struct{}{}
	}

	for _, t := range s.stagingTaskRegistry.Tasks() {
		all[t] = struct{}{}
	}

	return all
}

func (s *shutdownHandler) terminate() {
	s.logger.Infof("All instances and staging tasks stopped, exiting.")
	s.pidFile.Release()
	os.Exit(0)
}
