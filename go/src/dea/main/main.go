package main

import (
	"dea"
	"dea/boot"
	cfg "dea/config"
	ds "dea/directory_server"
	"dea/droplet"
	"dea/loggregator"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/responders"
	rtr "dea/router_client"
	"dea/staging"
	"dea/starting"
	"dea/utils"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/gorouter/common"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/localip"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"github.com/cloudfoundry/yagnats"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"
)

const (
	DEFAULT_HEARTBEAT_INTERVAL = 10 * time.Second
	DROPLET_REAPER_INTERVAL    = 60 * time.Second
	EXIT_REASON_STOPPED        = "STOPPED"
	EXIT_REASON_SHUTDOWN       = "DEA_SHUTDOWN"
	EXIT_REASON_EVACUATION     = "DEA_EVACUATION"
)

var signalsOfInterest = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}

type signalHandler interface {
	handleSignal(s os.Signal)
}

type Terminator interface {
	Shutdown()
	Terminate()
}

type bootstrap struct {
	config              *cfg.Config
	nats                *dea.Nats
	responders          []responders.Responder
	evacuationProcessed bool
	shutdown_processed  bool
	routerClient        rtr.RouterClient
	pidFile             *dea.PidFile
	instanceRegistry    starting.InstanceRegistry
	stagingTaskRegistry *staging.StagingTaskRegistry
	dropletRegistry     droplet.DropletRegistry
	logger              *steno.Logger
	component           *common.VcapComponent
	signalChannel       chan<- os.Signal
	localIp             string
	heartbeatTicker     *time.Ticker
	directoryServer     *ds.DirectoryServerV1
	directoryServerV2   *ds.DirectoryServerV2
	resource_manager    resmgr.ResourceManager
	registrationTicker  *time.Ticker
	signalHandler
	Terminator
	instanceManager boot.InstanceManager
	snapshot        boot.Snapshot
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config path>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	config, err := cfg.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config not found at '%s'\n%s", configPath, err.Error())
		os.Exit(1)
	}

	b := newBootstrap(&config)
	err = b.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed\n%s", configPath, err.Error())
		os.Exit(1)
	}

	b.Start()
}

func newBootstrap(c *cfg.Config) *bootstrap {
	if c.BaseDir == "" {
		panic("EMPTY")
	}
	b := &bootstrap{config: c}
	b.signalHandler = b
	b.Terminator = b
	return b
}

func (b *bootstrap) Config() *cfg.Config {
	return b.config
}

func (b *bootstrap) LocalIp() string {
	return b.localIp
}

func (b *bootstrap) DropletRegistry() droplet.DropletRegistry {
	return b.dropletRegistry
}

func (b *bootstrap) InstanceRegistry() starting.InstanceRegistry {
	return b.instanceRegistry
}

func (b *bootstrap) ResourceManager() resmgr.ResourceManager {
	return b.resource_manager
}

func (b *bootstrap) RouterClient() rtr.RouterClient {
	return b.routerClient
}

func (b *bootstrap) Snapshot() boot.Snapshot {
	return b.snapshot
}

func (b *bootstrap) InstanceManager() boot.InstanceManager {
	return b.instanceManager
}

func (b *bootstrap) StagingTaskRegistry() *staging.StagingTaskRegistry {
	return b.stagingTaskRegistry
}

func (b *bootstrap) Nats() yagnats.NATSClient {
	return b.nats.NatsClient
}

func (b *bootstrap) Setup() error {
	_, err := b.setupLogger()
	if err != nil {
		return err
	}

	if err := b.setupLoggregator(); err != nil {
		return err
	}

	b.setupRegistries()
	b.setupInstanceManager()
	b.setupSnapshot()

	b.resource_manager = resmgr.NewResourceManager(b.instanceRegistry, b.stagingTaskRegistry,
		&b.config.Resources)

	if err := b.setupDirectoryServers(); err != nil {
		return err
	}

	b.setupSignalHandlers()
	b.setupDirectories()
	b.setupPidFile()
	b.setupSweepers()

	b.setupNats()

	b.setupComponent()

	return nil
}

func (b *bootstrap) setupNats() error {
	nats, err := dea.NewNats(b.config.NatsServers)
	b.nats = nats
	return err
}

func (b *bootstrap) setupLoggregator() (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case error:
				err = r.(error)
			default:
				err = fmt.Errorf("%s", r)
			}
		}
	}()

	if b.config.Loggregator.Router != "" && b.config.Loggregator.SharedSecret != "" {
		index := strconv.FormatUint(uint64(b.config.Index), 10)

		var emit emitter.Emitter
		emit, err = emitter.NewEmitter(b.config.Loggregator.Router, "DEA",
			index, b.config.Loggregator.SharedSecret, b.logger)
		if err != nil {

			return
		}

		loggregator.SetEmitter(emit)

		emit, err = emitter.NewEmitter(b.config.Loggregator.Router, "STG",
			index, b.config.Loggregator.SharedSecret, b.logger)
		if err != nil {
			return
		}

		loggregator.SetStagingEmitter(emit)
	}

	return
}

func (b *bootstrap) setupRegistries() {
	b.dropletRegistry = droplet.NewDropletRegistry(path.Join(b.config.BaseDir, "droplets"))
	b.instanceRegistry = starting.NewInstanceRegistry(b.config)
	b.stagingTaskRegistry = staging.NewStagingTaskRegistry(staging.NewStagingTask)
}

func (b *bootstrap) setupInstanceManager() {
	b.instanceManager = boot.NewInstanceManager(b)
}

func (b *bootstrap) setupSnapshot() {
	b.snapshot = boot.NewSnapshot(b.instanceRegistry, b.stagingTaskRegistry, b.instanceManager, b.config.BaseDir)
}

func (b *bootstrap) setupDirectoryServers() error {
	var err error

	if b.localIp == "" {
		b.localIp, err = localip.LocalIP()
		if err != nil {
			return err
		}
	}

	b.directoryServer, err = ds.NewDirectoryServerV1(b.localIp, b.config.DirectoryServer.V1Port,
		ds.NewDirectory(b.instanceRegistry))
	if err != nil {
		return err
	}

	b.directoryServerV2, err = ds.NewDirectoryServerV2(b.localIp, b.config.Domain, b.config.DirectoryServer)
	if err != nil {
		return err
	}
	b.directoryServerV2.Configure_endpoints(b.instanceRegistry, b.stagingTaskRegistry)
	return nil
}

func (b *bootstrap) setupLogger() (*steno.Logger, error) {
	c := b.config.Logging
	l := steno.LOG_INFO

	if c.Level != "" {
		var err error
		l, err = steno.GetLogLevel(c.Level)
		if err != nil {
			return nil, err
		}
	}

	s := make([]steno.Sink, 0)
	if c.File != "" {
		s = append(s, steno.NewFileSink(c.File))
	} else {
		s = append(s, steno.NewIOSink(os.Stdout))
	}

	if c.Syslog != "" {
		s = append(s, steno.NewSyslogSink(c.Syslog))
	}

	stenoConfig := &steno.Config{
		Sinks: s,
		Codec: steno.NewJsonCodec(),
		Level: l,
	}

	steno.Init(stenoConfig)
	logger := steno.NewLogger("DEA")
	b.logger = logger

	logger.Info("Dea started")

	return logger, nil
}

func (b *bootstrap) setupDirectories() {
	dirs := []string{"db", "droplets", "instances", "tmp", "staging"}
	for _, d := range dirs {
		os.MkdirAll(path.Join(b.config.BaseDir, d), 0755)
	}

	os.MkdirAll(b.config.CrashesPath, 0755)
}

func (b *bootstrap) setupPidFile() error {
	pidFile, err := dea.NewPidFile(b.config.PidFile)
	if err != nil {
		return err
	}
	b.pidFile = pidFile
	return nil
}

func (b *bootstrap) setupSweepers() {
	// Heartbeats of instances we're managing
	hbInterval := b.config.Intervals.Heartbeat
	if hbInterval == 0 {
		hbInterval = DEFAULT_HEARTBEAT_INTERVAL
	}

	b.heartbeatTicker = utils.RepeatFixed(hbInterval, func() {
		b.SendHeartbeat()
	})

	// Ensure we keep around only the most recent crash for short amount of time
	b.instanceRegistry.StartReapers()

	utils.Repeat(DROPLET_REAPER_INTERVAL, func() {
		b.reapUnreferencedDroplets()
	})
}

func (b *bootstrap) stopSweepers() {
	if b.heartbeatTicker != nil {
		b.heartbeatTicker.Stop()
	}
}

func (b *bootstrap) SendHeartbeat() {
	instances := b.instanceRegistry.Instances()
	interested := make([]*starting.Instance, 0, 1)
	for _, i := range instances {
		switch i.State() {
		case starting.STATE_STARTING, starting.STATE_RUNNING, starting.STATE_CRASHED:
			interested = append(interested, i)
		}
	}
	if len(interested) > 0 {
		hbs := protocol.NewHeartbeatResponseV1(b.component.UUID, interested)
		bytes, err := json.Marshal(hbs)
		if err != nil {
			b.logger.Error(err.Error())
			return
		}
		b.nats.NatsClient.Publish("dea.heartbeat", bytes)
	}
}

func (b *bootstrap) setupSignalHandlers() {
	c := make(chan os.Signal, 1)
	b.signalChannel = c
	go func() {
		s := <-c
		b.signalHandler.handleSignal(s)
	}()
	signal.Notify(c, signalsOfInterest...)
}

func (b *bootstrap) handleSignal(s os.Signal) {
	switch s {
	case syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT:
		debug.PrintStack()
		b.Shutdown()
	case syscall.SIGUSR1:
		b.trap_usr1()
	case syscall.SIGUSR2:
		b.evacuate()
	}
}

func (b *bootstrap) trap_usr1() {
	b.ignoreSignals()
	b.sendShutdownMessage()
	for _, r := range b.responders {
		r.Stop()
	}
}

func (b *bootstrap) ignoreSignals() {
	signal.Stop(b.signalChannel)
}

func (b *bootstrap) Shutdown() {
	if b.shutdown_processed {
		return
	}

	b.shutdown_processed = true
	b.ignoreSignals()
	b.logger.Info("Shutting down")
	if !b.isEvacuating() {
		b.sendShutdownMessage()
	}

	for _, r := range b.responders {
		r.Stop()
	}

	b.nats.Stop()

	b.routerClient.Unregister_directory_server(b.localIp, b.directoryServerV2.Port(),
		b.directoryServerV2.External_hostname())

	for _, i := range b.instanceRegistry.Instances() {
		i.Stop()
	}
	for _, t := range b.stagingTaskRegistry.Tasks() {
		t.Stop()
	}

	b.logger.Info("All instances and staging tasks stopped, exiting.")
	// Terminate after nats sends all queued messages
	b.nats.Stop()
	b.Terminator.Terminate()
}

func (b *bootstrap) sendShutdownMessage() {
	gbm := protocol.NewGoodbyeMessage(b.component.UUID, b.localIp, b.instanceRegistry.AppIdToCount())
	bytes, err := json.Marshal(gbm)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}
	b.nats.NatsClient.Publish("dea.shutdown", bytes)
}

func (b *bootstrap) SendExitMessage(i *starting.Instance, reason string) {
	exitm := protocol.NewExitMessage(*i, reason)
	bytes, err := json.Marshal(exitm)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}
	b.nats.NatsClient.Publish("droplet.exited", bytes)
}

func (b *bootstrap) SendInstanceStopMessage(instance *starting.Instance) {
	// This is a little wonky but ensures that we don't send an exited
	// message twice. During evacuation, an exit message is sent for each
	// running app, the evacuation interval is allowed to pass, and the app
	// is finally stopped.  This allows the app to be started on another DEA
	// and begin serving traffic before we stop it here.
	if b.evacuationProcessed {
		return
	}

	exitMessage := EXIT_REASON_STOPPED
	if b.shutdown_processed {
		exitMessage = EXIT_REASON_SHUTDOWN
	}
	b.SendExitMessage(instance, exitMessage)
}

func (b *bootstrap) evacuate() {
	if b.evacuationProcessed {
		b.logger.Info("Evacuation already processed, doing nothing.")
		return
	}

	b.evacuationProcessed = true
	b.logger.Info("Evacuating apps")

	b.sendShutdownMessage()

	b.ignoreSignals()
	b.stopSweepers()

	for _, r := range b.responders {
		r.Stop()
	}

	if b.instanceRegistry != nil {
		for _, i := range b.instanceRegistry.Instances() {
			state := i.State()
			if state == starting.STATE_RUNNING || state == starting.STATE_STARTING {
				b.SendExitMessage(i, EXIT_REASON_EVACUATION)
			}
		}
	}

	time.AfterFunc(b.config.EvacuationDelay, func() {
		b.Terminator.Shutdown()
	})
}

func (b *bootstrap) Terminate() {
	b.pidFile.Release()
	os.Exit(0)
}

func (b *bootstrap) Start() {
	b.snapshot.Load()

	b.startComponent()

	b.startNats()

	b.directoryServer.Start()

	b.setupRouterClient()
	b.greetRouter()

	b.routerClient.Register_directory_server(
		b.localIp, b.directoryServerV2.Port(),
		b.directoryServerV2.External_hostname())
	b.directoryServerV2.Start()

	b.start_finish()
}

func (b *bootstrap) start_finish() {
	bytes, err := json.Marshal(protocol.NewHelloMessage(b.component.UUID,
		b.localIp, b.directoryServer.Port))
	if err != nil {
		b.logger.Error(err.Error())
		return
	}

	b.nats.NatsClient.Publish("dea.start", bytes)
	for _, r := range b.responders {
		if lr, ok := r.(responders.LocatorResponder); ok {
			lr.Advertise()
		}
	}

	instances := b.instanceRegistry.Instances()
	if len(instances) > 0 {
		b.logger.Infof("Loaded %d instances from snapshot", len(instances))
		b.SendHeartbeat()
	}
}

func (b *bootstrap) setupRouterClient() {
	b.routerClient = rtr.NewRouterClient(b.config, b.nats, b.component.UUID, b.localIp)
}

func (b *bootstrap) startComponent() error {
	common.StartComponent(b.component)
	common.Register(b.component, b.nats.NatsClient)

	return nil
}

func (b *bootstrap) startNats() {
	if err := b.nats.Start(b); err != nil {
		panic(err)
	}

	b.responders = []responders.Responder{
		responders.NewDeaLocator(b.nats.NatsClient, b.component.UUID, b.resource_manager, b.config),
		responders.NewStagingLocator(b.nats.NatsClient, b.component.UUID, b.resource_manager, b.config),
		responders.NewStaging(b, b.component.UUID, b.directoryServerV2),
	}

	for _, r := range b.responders {
		r.Start()
	}
}

func (b *bootstrap) greetRouter() {
	b.routerClient.Greet(func(msg *yagnats.Message) {
		b.HandleRouterStart(msg)
	})
}

func (b *bootstrap) setupComponent() *time.Timer {
	varz := &common.Varz{}
	varz.UniqueVarz = map[string]interface{}{
		"stacks": b.config.Stacks,
	}

	statusConfig := b.config.Status
	component := &common.VcapComponent{
		Type:        "DEA",
		Index:       b.config.Index,
		Credentials: []string{statusConfig.User, statusConfig.Password},
		Varz:        varz,
		Healthz:     &common.Healthz{},
		Logger:      b.logger,
		Host:        fmt.Sprintf("%s:%d", b.localIp, statusConfig.Port),
	}

	b.component = component

	return utils.Repeat(DEFAULT_HEARTBEAT_INTERVAL, b.periodic_varz_update)
}

func (b *bootstrap) isEvacuating() bool {
	return b.evacuationProcessed
}

func (b *bootstrap) reapUnreferencedDroplets() {
	instance_registry_shas := make(map[string]string)
	for _, instance := range b.instanceRegistry.Instances() {
		sha := instance.DropletSHA1()
		instance_registry_shas[sha] = sha
	}

	staging_registry_shas := make(map[string]string)
	for _, task := range b.stagingTaskRegistry.Tasks() {
		sha := task.DropletSHA1()
		staging_registry_shas[sha] = sha
	}

	all_shas := b.dropletRegistry.SHA1s()
	for _, sha := range all_shas {
		if _, exists := instance_registry_shas[sha]; exists {
			continue
		}
		if _, exists := staging_registry_shas[sha]; exists {
			continue
		}

		b.logger.Debugf("Removing droplet for sha=%s", sha)
		droplet := b.dropletRegistry.Remove(sha)
		droplet.Destroy()
	}
}

func (b *bootstrap) HandleHealthManagerStart(msg *yagnats.Message) {
	b.SendHeartbeat()
}

func (b *bootstrap) HandleRouterStart(msg *yagnats.Message) {
	routerStart := &common.RouterStart{}
	if err := json.Unmarshal(msg.Payload, routerStart); err != nil {
		b.logger.Errorf("Invalid Router Start payload: %s", msg.Payload)
		return
	}

	interval := time.Duration(routerStart.MinimumRegisterIntervalInSeconds) * time.Second

	b.register_routes()

	if interval > 0 {
		if b.registrationTicker != nil {
			b.registrationTicker.Stop()
		}
		utils.RepeatFixed(interval, func() { b.register_routes() })
	}

}

func (b *bootstrap) register_routes() {
	for _, i := range b.instanceRegistry.Instances() {
		if i.State() == starting.STATE_RUNNING || len(i.ApplicationUris()) > 0 {
			b.routerClient.RegisterInstance(i, nil)
		}
	}
}

func (b *bootstrap) HandleDeaStatus(msg *yagnats.Message) {
	response := protocol.NewDeaStatusResponse(b.component.UUID, b.localIp, b.directoryServer.Port,
		b.resource_manager.MemoryCapacity(),
		uint(b.resource_manager.ReservedMemory()), uint(b.resource_manager.UsedMemory()))
	if bytes, err := json.Marshal(response); err == nil {
		b.nats.NatsClient.Publish(msg.ReplyTo, bytes)
	} else {
		b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
	}
}

func (b *bootstrap) HandleDeaDirectedStart(msg *yagnats.Message) {
	d := map[string]interface{}{}
	if err := json.Unmarshal(msg.Payload, &d); err == nil {
		b.instanceManager.StartApp(d)
	} else {
		b.logger.Errorf("HandleDeaDirectedStart: marshal failed, %s", err.Error())
	}

}
func (b *bootstrap) HandleDeaStop(msg *yagnats.Message) {
	var d map[string]interface{}
	if err := json.Unmarshal(msg.Payload, d); err == nil {
		b.instanceRegistry.Instances_filtered_by_message(d, func(i *starting.Instance) {
			switch i.State() {
			case starting.STATE_RUNNING, starting.STATE_STARTING:
			default:
				return
			}
			if err := i.Stop(); err != nil {
				b.logger.Warnf("Failed stopping %s: %s", i, err.Error())
			}
		})
	}
}

func (b *bootstrap) HandleDeaUpdate(msg *yagnats.Message) {
	var d map[string]interface{}
	if err := json.Unmarshal(msg.Payload, d); err != nil {
		b.logger.Errorf("HandleDeaUpdate: marshal failed, %s", err.Error())
	}
	app_id := d["droplet"].(string)
	uris := d["uris"].([]string)
	app_version := d["version"].(string)

	for _, i := range b.instanceRegistry.InstancesForApplication(app_id) {

		current_uris := i.ApplicationUris()

		b.logger.Debug("Mapping new URIs")
		b.logger.Debugf("New: %v Old: %v", uris, current_uris)

		new_uris := utils.Difference(uris, current_uris)
		if len(new_uris) > 0 {
			b.routerClient.RegisterInstance(i, map[string]interface{}{"uris": new_uris})
		}

		obsolete_uris := utils.Difference(current_uris, uris)
		if len(obsolete_uris) > 0 {
			b.routerClient.UnregisterInstance(i, map[string]interface{}{"uris": obsolete_uris})
		}

		i.SetApplicationUris(uris)
		if app_version != "" {
			i.SetApplicationVersion(app_version)
			b.instanceRegistry.ChangeInstanceId(i)
		}
	}
}

func (b *bootstrap) HandleDeaFindDroplet(msg *yagnats.Message) {
	var d map[string]interface{}
	if err := json.Unmarshal(msg.Payload, d); err == nil {
		b.instanceRegistry.Instances_filtered_by_message(d, func(i *starting.Instance) {
			response := protocol.NewFindDropletResponse(b.UUID(), b.localIp, i, b.directoryServer, b.directoryServerV2, d)
			if bytes, err := json.Marshal(response); err == nil {
				b.nats.NatsClient.Publish(msg.ReplyTo, bytes)
			} else {
				b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
			}
		})
	}
}

func (b *bootstrap) UUID() string {
	return b.component.UUID
}

func (b *bootstrap) periodic_varz_update() {
	mem_required := b.config.Staging.MemoryLimitMB
	disk_required := b.config.Staging.DiskLimitMB
	reservable_stagers := b.resource_manager.NumberReservable(mem_required, disk_required)
	available_memory_ratio := b.resource_manager.AvailableMemoryRatio()
	available_disk_ratio := b.resource_manager.AvailableDiskRatio()

	b.component.Varz.Lock()
	defer b.component.Varz.Unlock()

	d := b.component.Varz.UniqueVarz.(map[string]interface{})

	stagers := 0
	if reservable_stagers > 0 {
		stagers = 1
	}
	d["can_stage"] = stagers
	d["reservable_stagers"] = reservable_stagers
	d["available_memory_ratio"] = available_memory_ratio
	d["available_disk_ratio"] = available_disk_ratio
	d["instance_registry"] = b.instanceRegistry.ToHash()
}
