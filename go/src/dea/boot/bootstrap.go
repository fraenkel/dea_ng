package boot

import (
	"dea"
	cfg "dea/config"
	ds "dea/directory_server"
	"dea/droplet"
	"dea/lifecycle"
	"dea/loggregator"
	"dea/nats"
	"dea/protocol"
	resmgr "dea/resource_manager"
	"dea/responders"
	rtr "dea/router_client"
	"dea/snapshot"
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
	"path"
	"strconv"
	"time"
)

const (
	DEFAULT_HEARTBEAT_INTERVAL = 10 * time.Second
	DROPLET_REAPER_INTERVAL    = 60 * time.Second
)

type Bootstrap struct {
	config              *cfg.Config
	nats                *nats.Nats
	responders          []dea.Responder
	routerClient        dea.RouterClient
	pidFile             *dea.PidFile
	instanceRegistry    dea.InstanceRegistry
	stagingTaskRegistry dea.StagingTaskRegistry
	dropletRegistry     dea.DropletRegistry
	logger              *steno.Logger
	component           *common.VcapComponent
	localIp             string
	heartbeatTicker     *time.Ticker
	directoryServerV2   *ds.DirectoryServerV2
	resource_manager    dea.ResourceManager
	registrationTicker  *time.Ticker
	signalHandler       lifecycle.SignalHandler
	instanceManager     dea.InstanceManager
	snapshot            dea.Snapshot
}

func NewBootstrap(c *cfg.Config) *Bootstrap {
	if c.BaseDir == "" {
		panic("EMPTY")
	}
	b := &Bootstrap{config: c}
	return b
}

func (b *Bootstrap) Config() *cfg.Config {
	return b.config
}

func (b *Bootstrap) LocalIp() string {
	return b.localIp
}

func (b *Bootstrap) DropletRegistry() dea.DropletRegistry {
	return b.dropletRegistry
}

func (b *Bootstrap) InstanceRegistry() dea.InstanceRegistry {
	return b.instanceRegistry
}

func (b *Bootstrap) ResourceManager() dea.ResourceManager {
	return b.resource_manager
}

func (b *Bootstrap) RouterClient() dea.RouterClient {
	return b.routerClient
}

func (b *Bootstrap) Snapshot() dea.Snapshot {
	return b.snapshot
}

func (b *Bootstrap) InstanceManager() dea.InstanceManager {
	return b.instanceManager
}

func (b *Bootstrap) StagingTaskRegistry() dea.StagingTaskRegistry {
	return b.stagingTaskRegistry
}

func (b *Bootstrap) Nats() yagnats.NATSClient {
	return b.nats.Client()
}

func (b *Bootstrap) Setup() error {
	_, err := b.setupLogger()
	if err != nil {
		return err
	}

	if err := b.setupLoggregator(); err != nil {
		return err
	}

	b.setupNats()

	b.setupRegistries()
	b.setupInstanceManager()
	b.setupSnapshot()

	b.resource_manager = resmgr.NewResourceManager(b.instanceRegistry, b.stagingTaskRegistry,
		&b.config.Resources)

	if err := b.setupDirectoryServers(); err != nil {
		return err
	}

	b.setupDirectories()
	b.setupPidFile()
	b.setupSweepers()

	b.setupComponent()
	b.setupRouterClient()

	return nil
}

func (b *Bootstrap) setupNats() error {
	nats, err := nats.NewNats(b.config.NatsServers)
	b.nats = nats
	return err
}

func (b *Bootstrap) setupLoggregator() (err error) {
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

func (b *Bootstrap) setupRegistries() {
	b.dropletRegistry = droplet.NewDropletRegistry(path.Join(b.config.BaseDir, "droplets"))
	b.instanceRegistry = starting.NewInstanceRegistry(b.config)
	b.stagingTaskRegistry = staging.NewStagingTaskRegistry(b.config, b.dropletRegistry, staging.NewStagingTask)
}

func (b *Bootstrap) setupInstanceManager() {
	b.instanceManager = starting.NewInstanceManager(b)
}

func (b *Bootstrap) setupSnapshot() {
	b.snapshot = snapshot.NewSnapshot(b.instanceRegistry, b.stagingTaskRegistry, b.instanceManager, b.config.BaseDir)
}

func (b *Bootstrap) setupDirectoryServers() error {
	var err error

	if b.localIp == "" {
		b.localIp, err = localip.LocalIP()
		if err != nil {
			return err
		}
	}

	b.directoryServerV2, err = ds.NewDirectoryServerV2(b.localIp, b.config.Domain, b.routerClient, b.config.DirectoryServer)
	if err != nil {
		return err
	}
	b.directoryServerV2.Configure_endpoints(b.instanceRegistry, b.stagingTaskRegistry)
	return nil
}

func (b *Bootstrap) setupLogger() (*steno.Logger, error) {
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

func (b *Bootstrap) setupDirectories() {
	dirs := []string{"db", "droplets", "instances", "tmp", "staging"}
	for _, d := range dirs {
		os.MkdirAll(path.Join(b.config.BaseDir, d), 0755)
	}

	os.MkdirAll(b.config.CrashesPath, 0755)
}

func (b *Bootstrap) setupPidFile() error {
	pidFile, err := dea.NewPidFile(b.config.PidFile)
	if err != nil {
		return err
	}
	b.pidFile = pidFile
	return nil
}

func (b *Bootstrap) setupSweepers() {
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

func (b *Bootstrap) stopSweepers() {
	if b.heartbeatTicker != nil {
		b.heartbeatTicker.Stop()
	}
}

func (b *Bootstrap) SendHeartbeat() {
	instances := b.instanceRegistry.Instances()
	interested := make([]dea.Instance, 0, 1)
	for _, i := range instances {
		switch i.State() {
		case dea.STATE_STARTING, dea.STATE_RUNNING,
			dea.STATE_CRASHED, dea.STATE_EVACUATING:
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
		b.nats.Client().Publish("dea.heartbeat", bytes)
	}
}

func (b *Bootstrap) Start() {
	b.snapshot.Load()

	b.startComponent()

	b.startNats()

	b.greetRouter()

	b.directoryServerV2.Start()

	b.startVarz()

	b.setupHandlers()
	b.start_finish()
}

func (b *Bootstrap) start_finish() {
	bytes, err := json.Marshal(protocol.NewHelloMessage(b.component.UUID, b.localIp))
	if err != nil {
		b.logger.Error(err.Error())
		return
	}
	b.nats.Client().Publish("dea.start", bytes)

	for _, r := range b.responders {
		if lr, ok := r.(dea.LocatorResponder); ok {
			lr.Advertise()
		}
	}

	instances := b.instanceRegistry.Instances()
	if len(instances) > 0 {
		b.logger.Infof("Loaded %d instances from snapshot", len(instances))
		b.SendHeartbeat()
	}
}

func (b *Bootstrap) setupHandlers() {
	evac := lifecycle.NewEvacuationHandler(b.config.EvacuationBailOut, b.responders, b.instanceRegistry,
		b.nats.Client(), b.logger)
	shutdown := lifecycle.NewShutdownHandler(b.responders, b.instanceRegistry, b.stagingTaskRegistry, b.dropletRegistry,
		b.directoryServerV2, b.nats, b, b.pidFile, b.logger)

	b.signalHandler = lifecycle.NewSignalHandler(b.UUID(), b.localIp, b.responders, evac, shutdown,
		b.instanceRegistry, b.nats.Client(), b.logger)
	b.signalHandler.Setup()
}

func (b *Bootstrap) setupRouterClient() {
	b.routerClient = rtr.NewRouterClient(b.config, b.nats, b.component.UUID, b.localIp)
}

func (b *Bootstrap) startComponent() error {
	common.StartComponent(b.component)
	common.Register(b.component, b.nats.Client())

	return nil
}

func (b *Bootstrap) startNats() {
	if err := b.nats.Start(b); err != nil {
		panic(err)
	}

	b.responders = []dea.Responder{
		responders.NewDeaLocator(b.nats.Client(), b.component.UUID, b.resource_manager, b.config),
		responders.NewStagingLocator(b.nats.Client(), b.component.UUID, b.resource_manager, b.config),
		responders.NewStaging(b, b.component.UUID, b.directoryServerV2),
	}

	for _, r := range b.responders {
		r.Start()
	}
}

func (b *Bootstrap) startVarz() *time.Timer {
	return utils.Repeat(DEFAULT_HEARTBEAT_INTERVAL, b.periodic_varz_update)
}

func (b *Bootstrap) greetRouter() {
	b.routerClient.Greet(func(msg *yagnats.Message) {
		b.HandleRouterStart(msg)
	})
}

func (b *Bootstrap) setupComponent() {
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
}

func (b *Bootstrap) reapUnreferencedDroplets() {
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

func (b *Bootstrap) HandleHealthManagerStart(msg *yagnats.Message) {
	b.SendHeartbeat()
}

func (b *Bootstrap) HandleRouterStart(msg *yagnats.Message) {
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

func (b *Bootstrap) register_routes() {
	for _, i := range b.instanceRegistry.Instances() {
		state := i.State()
		if state == dea.STATE_RUNNING || state == dea.STATE_EVACUATING || len(i.ApplicationUris()) > 0 {
			b.routerClient.RegisterInstance(i, nil)
		}
	}
}

func (b *Bootstrap) HandleDeaStatus(msg *yagnats.Message) {
	response := protocol.NewDeaStatusResponse(b.component.UUID, b.localIp,
		b.resource_manager.MemoryCapacity(),
		uint(b.resource_manager.ReservedMemory()), uint(b.resource_manager.UsedMemory()))
	if bytes, err := json.Marshal(response); err == nil {
		b.nats.Client().Publish(msg.ReplyTo, bytes)
	} else {
		b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
	}
}

func (b *Bootstrap) HandleDeaDirectedStart(msg *yagnats.Message) {
	d := map[string]interface{}{}
	if err := json.Unmarshal(msg.Payload, &d); err == nil {
		b.instanceManager.StartApp(d)
	} else {
		b.logger.Errorf("HandleDeaDirectedStart: marshal failed, %s", err.Error())
	}

}
func (b *Bootstrap) HandleDeaStop(msg *yagnats.Message) {
	var d map[string]interface{}
	if err := json.Unmarshal(msg.Payload, d); err == nil {
		b.instanceRegistry.Instances_filtered_by_message(d, func(i dea.Instance) {
			switch i.State() {
			case dea.STATE_RUNNING, dea.STATE_STARTING, dea.STATE_EVACUATING:
			default:
				return
			}
			i.Stop(func(err error) error {
				if err != nil {
					b.logger.Warnf("Failed stopping %s: %s", i, err.Error())
				}
				return nil
			})
		})
	}
}

func (b *Bootstrap) HandleDeaUpdate(msg *yagnats.Message) {
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

func (b *Bootstrap) HandleDeaFindDroplet(msg *yagnats.Message) {
	var d map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &d); err == nil {
		b.instanceRegistry.Instances_filtered_by_message(d, func(i dea.Instance) {
			response := protocol.NewFindDropletResponse(b.UUID(), b.localIp, i, b.directoryServerV2, d)
			if bytes, err := json.Marshal(response); err == nil {
				b.nats.Client().Publish(msg.ReplyTo, bytes)
			} else {
				b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
			}
		})
	}
}

func (b *Bootstrap) UUID() string {
	return b.component.UUID
}
func (b *Bootstrap) Terminate() {
	os.Exit(0)
}

func (b *Bootstrap) periodic_varz_update() {
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
