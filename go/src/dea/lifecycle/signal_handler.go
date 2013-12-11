package lifecycle

import (
	"dea"
	"dea/protocol"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

var signalsOfInterest = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}

type SignalHandler interface {
	Setup()
}

type signalHandler struct {
	signalChannel     chan<- os.Signal
	uuid              string
	localIp           string
	responders        []dea.Responder
	evacuationHandler EvacuationHandler
	shutdownHandler   ShutdownHandler
	instanceRegistry  dea.InstanceRegistry
	natsClient        yagnats.NATSClient
	logger            *steno.Logger
}

func NewSignalHandler(uuid, localIp string, responders []dea.Responder, evac EvacuationHandler, shutdownHandler ShutdownHandler,
	iRegistry dea.InstanceRegistry, natsClient yagnats.NATSClient, logger *steno.Logger) SignalHandler {
	return &signalHandler{
		uuid:              uuid,
		localIp:           localIp,
		responders:        responders,
		evacuationHandler: evac,
		shutdownHandler:   shutdownHandler,
		instanceRegistry:  iRegistry,
		natsClient:        natsClient,
		logger:            logger,
	}
}

func (sh *signalHandler) Setup() {
	c := make(chan os.Signal, 1)
	sh.signalChannel = c
	go func() {
		s := <-c
		sh.logger.Warnf("caught signal %s", s.String())
		sh.handleSignal(s)
	}()
	signal.Notify(c, signalsOfInterest...)
}

func (sh *signalHandler) handleSignal(s os.Signal) {
	switch s {
	case syscall.SIGQUIT:
		debug.PrintStack()
		sh.shutdown()
	case syscall.SIGTERM, syscall.SIGINT:
		sh.shutdown()
	case syscall.SIGUSR1:
		sh.trap_usr1()
	case syscall.SIGUSR2:
		sh.trap_usr2()
	}
}

func (sh *signalHandler) trap_usr1() {
	sendGoodbyeMessage(sh.goodbyeMsg(), sh.natsClient, sh.logger)
	stopResponders(sh.responders)
}

func (sh *signalHandler) trap_usr2() {
	if canShutdown := sh.evacuationHandler.Evacuate(sh.goodbyeMsg()); canShutdown {
		sh.shutdown()
	}
}

func (sh *signalHandler) shutdown() {
	sh.shutdownHandler.Shutdown(sh.goodbyeMsg())
}

func (sh *signalHandler) ignoreSignals() {
	signal.Stop(sh.signalChannel)
}

func (sh *signalHandler) goodbyeMsg() *protocol.GoodbyeMessage {
	return protocol.NewGoodbyeMessage(sh.uuid, sh.localIp, sh.instanceRegistry.AppIdToCount())
}
