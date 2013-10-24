package main

import (
	"dea"
	"dea/config"
	ds "dea/directory_server"
	"dea/droplet"
	"dea/protocol"
	"dea/responders"
	"dea/staging"
	"dea/starting"
	"dea/utils"
	"encoding/json"
	"flag"
	"fmt"
	cfmessage "github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/gorouter/common"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/localip"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"
)

const (
	DEFAULT_HEARTBEAT_INTERVAL   = 10 //# In secs
	DROPLET_REAPER_INTERVAL_SECS = 10
	EXIT_REASON_STOPPED          = "STOPPED"
	EXIT_REASON_CRASHED          = "CRASHED"
	EXIT_REASON_SHUTDOWN         = "DEA_SHUTDOWN"
	EXIT_REASON_EVACUATION       = "DEA_EVACUATION"
)

var signalsOfInterest = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}

type bootstrap struct {
	config              *config.Config
	nats                *dea.Nats
	responders          []responders.Responder
	evacuationProcessed bool
	shutdown_processed  bool
	routerClient        dea.RouterClient
	varz                *common.Varz
	pidFile             *dea.PidFile
	instanceRegistry    *starting.InstanceRegistry
	stagingTaskRegistry *staging.StagingTaskRegistry
	dropletRegistry     *droplet.DropletRegistry
	logger              *steno.Logger
	emitter             emitter.Emitter
	component           *common.VcapComponent
	signalChannel       chan<- os.Signal
	localIp             string
	heartbeatTicker     *time.Ticker
	directoryServer     *ds.DirectoryServerV1
	directoryServerV2   *ds.DirectoryServerV2
	resource_manager    *dea.ResourceManager
	registrationTicker  *time.Ticker
}

func main() {
	var configPath string
	flag.StringVar(&configPath,
		"conf",
		"", "Path of the YAML configuration of the co-located DEA.")
	flag.Parse()

	config, err := config.ConfigFromFile(configPath)
	if err != nil {
		panic(err.Error())
	}

	bootstrap := bootstrap{config: config}
	err = bootstrap.setup()
	if err != nil {
		panic(err.Error())
	}

	bootstrap.start()
}

func (b bootstrap) setup() error {
	config := b.config
	logger, err := setupLogger(config.Logging)
	if err != nil {
		return err
	}
	b.logger = logger

	if config.Loggregator.Router != "" {
		emitter, err := emitter.NewEmitter(config.Loggregator.Router, "DEA",
			strconv.FormatUint(uint64(config.Index), 10), logger)
		if err != nil {
			return err
		}
		b.emitter = emitter
	}

	b.dropletRegistry = droplet.NewDropletRegistry(path.Join(config.BaseDir, "droplets"))
	b.instanceRegistry = starting.NewInstanceRegistry(config)
	b.stagingTaskRegistry = staging.NewStagingTaskRegistry()

	b.resource_manager = dea.NewResourceManager(b.instanceRegistry, b.stagingTaskRegistry,
		&b.config.Resources)

	localIp, err := localip.LocalIP()
	if err != nil {
		return err
	}
	b.localIp = localIp

	b.directoryServer, err = ds.NewDirectoryServerV1(localIp, config.DirectoryServer.V1Port,
		b.instanceRegistry)
	if err != nil {
		return err
	}

	b.directoryServerV2, err = ds.NewDirectoryServerV2(config.Domain, config.DirectoryServer.V2Port, config.DirectoryServer)
	if err != nil {
		return err
	}
	b.directoryServerV2.Configure_endpoints(b.instanceRegistry, b.stagingTaskRegistry)

	b.setupSignalHandlers()
	b.setupPidFile()
	b.setupSweepers()

	nats, err := dea.NewNats(b.config.NatsConfig)
	if err != nil {
		return err
	}
	b.nats = nats

	return nil
}

func setupLogger(c config.LoggingConfig) (*steno.Logger, error) {
	l, err := steno.GetLogLevel(c.Level)
	if err != nil {
		return nil, err
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
	return steno.NewLogger("DEA"), nil
}

func setupDirectories(config *config.Config) {
	dirs := []string{"db", "droplets", "instances", "tmp", "staging"}
	for _, d := range dirs {
		os.MkdirAll(path.Join(config.BaseDir, d), 0755)
	}

	os.MkdirAll(path.Join(config.BaseDir, "crashes"), 0755)
}

func (bootstrap *bootstrap) setupPidFile() error {
	pidFile, err := dea.NewPidFile(bootstrap.config.PidFile)
	if err != nil {
		return err
	}
	bootstrap.pidFile = pidFile
	return nil
}

func (b *bootstrap) setupSweepers() {
	// Heartbeats of instances we're managing
	hbInterval, exist := b.config.Intervals["heartbeat"]
	if !exist {
		hbInterval = DEFAULT_HEARTBEAT_INTERVAL
	}

	b.heartbeatTicker = utils.Repeat(func() {
		b.sendHeartbeat(b.instanceRegistry.Instances())
	}, time.Duration(hbInterval)*time.Second)

	// Ensure we keep around only the most recent crash for short amount of time
	b.instanceRegistry.StartReapers()

	utils.Repeat(func() {
		b.reapUnreferencedDroplets()
	}, DROPLET_REAPER_INTERVAL_SECS*time.Second)
}

func (b *bootstrap) stopSweepers() {
	b.heartbeatTicker.Stop()
}

func (b *bootstrap) sendHeartbeat(instances []*starting.Instance) {
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
		b.nats.MessageBus.Publish("dea.heartbeat", bytes)
	}
}

func (b *bootstrap) setupSignalHandlers() {
	c := make(chan os.Signal, 1)
	b.signalChannel = c
	go func() {
		s := <-c
		close(c)

		switch s {
		case syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT:
			b.shutdown()
		case syscall.SIGUSR1:
			b.terminate()
		case syscall.SIGUSR2:
			b.evacuate()
		}
	}()
	signal.Notify(c, signalsOfInterest...)
}

func (b *bootstrap) ignoreSignals() {
	close(b.signalChannel)
	b.signalChannel = nil
	c := make(chan os.Signal, 1)

	go func() {
		for {
			s := <-c
			b.logger.Warn("Caught SIG" + s.String() + ", ignoring.")
		}
	}()
	signal.Notify(c, signalsOfInterest...)
}

func (b *bootstrap) shutdown() {
	if b.shutdown_processed {
		return
	}

	b.shutdown_processed = true
	b.logger.Info("Shutting down")
	if !b.isEvacuating() {
		b.sendShutdownMessage()
	}

	close(b.signalChannel)

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
	b.terminate()

}

func (b *bootstrap) sendShutdownMessage() {
	gbm := protocol.NewGoodbyeMessage(b.component.UUID, b.localIp, b.instanceRegistry.AppIdToCount())
	bytes, err := json.Marshal(gbm)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}
	b.nats.MessageBus.Publish("dea.shutdown", bytes)
}

func (b *bootstrap) sendExitMessage(i *starting.Instance, reason string) {
	exitm := protocol.NewExitMessage(*i, reason)
	bytes, err := json.Marshal(exitm)
	if err != nil {
		b.logger.Error(err.Error())
		return
	}
	b.nats.MessageBus.Publish("droplet.exited", bytes)
}

func (b *bootstrap) send_instance_stop_message(instance *starting.Instance) {
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
	b.sendExitMessage(instance, exitMessage)
}

func (b *bootstrap) evacuate() {
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
				b.sendExitMessage(i, EXIT_REASON_EVACUATION)
			}
		}
	}

	time.AfterFunc(b.config.EvacuationDelay*time.Second, func() {
		b.shutdown()
	})
}

func (b *bootstrap) terminate() {
	b.pidFile.Release()
	os.Exit(0)
}

func (b *bootstrap) start() {
	b.startComponent()
	b.routerClient = dea.NewRouterClient(b.config, b.nats, b.component.UUID, b.localIp)
	b.startNats()
	b.directoryServer.Start()

	b.greetRouter()

	b.routerClient.Register_directory_server(
		b.localIp, b.directoryServerV2.Port(),
		b.directoryServerV2.External_hostname())
	b.directoryServerV2.Start()

	b.setupVarz()

	bytes, err := json.Marshal(protocol.NewHelloMessage(b.component.UUID,
		b.localIp, b.directoryServer.Port))
	if err != nil {
		b.logger.Error(err.Error())
		return
	}

	b.nats.MessageBus.Publish("dea.start", bytes)
	for _, r := range b.responders {
		if lr, ok := r.(responders.LocatorResponder); ok {
			lr.Advertise()
		}
	}
}

func (b *bootstrap) startComponent() error {
	statusConfig := b.config.Status
	component := &common.VcapComponent{
		Type:        "DEA",
		Index:       b.config.Index,
		Credentials: []string{statusConfig.User, statusConfig.Password},
		Varz:        b.varz,
		Healthz:     &common.Healthz{},
		Logger:      b.logger,
	}
	host, err := localip.LocalIP()
	if err != nil {
		return err
	}
	component.Host = fmt.Sprintf("%s:%d", host, statusConfig.Port)

	common.StartComponent(component)
	common.Register(component, b.nats.MessageBus)

	b.component = component
	return nil
}

func (b *bootstrap) startNats() {
	b.nats.Start(b)

	b.responders = []responders.Responder{
		responders.NewDeaLocator(b.nats.MessageBus, b.component.UUID, b.resource_manager, b.config),
		responders.NewStagingLocator(b.nats.MessageBus, b.component.UUID, b.resource_manager, b.config),
		responders.NewStaging(b, b.nats.MessageBus, b.component.UUID, b.stagingTaskRegistry, b.emitter, b.config),
	}

	for _, r := range b.responders {
		r.Start()
	}
}

func (b *bootstrap) greetRouter() {
	b.routerClient.Greet(func(response []byte) {
		b.HandleRouterStart(response)
	})
}

func (b *bootstrap) setupVarz() {
	varz := &common.Varz{}
	varz.UniqueVarz = map[string][]string{
		"stacks": b.config.Stacks,
	}

	b.varz = varz

	utils.Repeat(b.periodic_varz_update, DEFAULT_HEARTBEAT_INTERVAL*time.Second)
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

func (b *bootstrap) StartApp(data map[string]interface{}) {
	instance := b.create_instance(data)
	if instance != nil {
		instance.Start(nil)
	}
}

func (b *bootstrap) create_instance(attributes map[string]interface{}) *starting.Instance {
	limits := attributes["limits"].(map[string]uint64)
	memory := limits["mem"]
	disk := limits["disk"]
	if !b.resource_manager.CanReserve(memory, disk) {
		b.logger.Errorf("Unable to start instance: %s for app: %s, not enough resources available.", attributes["instance_index"], attributes["application_id"])
		return nil
	}

	instance := starting.NewInstance(attributes, b.config, b.dropletRegistry, b.localIp)
	instance.Setup()

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_CRASHED}, func() {
		b.sendExitMessage(instance, EXIT_REASON_CRASHED)
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_RUNNING}, func() {
		//        Notify others immediately
		b.sendHeartbeat([]*starting.Instance{instance})

		// Register with router
		b.routerClient.RegisterInstance(instance, nil)
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_CRASHED}, func() {
		b.routerClient.UnregisterInstance(instance, nil)
		b.sendExitMessage(instance, EXIT_REASON_CRASHED)
	})

	instance.On(starting.Transition{starting.STATE_RUNNING, starting.STATE_STOPPING}, func() {
		b.routerClient.UnregisterInstance(instance, nil)
		b.send_instance_stop_message(instance)
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_STOPPING}, func() {
		b.send_instance_stop_message(instance)
	})

	instance.On(starting.Transition{starting.STATE_STARTING, starting.STATE_RUNNING}, func() {
		b.save_snapshot()
	})

	instance.On(starting.Transition{starting.STATE_RUNNING, starting.STATE_STOPPING}, func() {
		b.save_snapshot()
	})

	instance.On(starting.Transition{starting.STATE_RUNNING, starting.STATE_CRASHED}, func() {
		b.save_snapshot()
	})

	instance.On(starting.Transition{starting.STATE_STOPPING, starting.STATE_STOPPED}, func() {
		b.instanceRegistry.Unregister(instance)
		go instance.Destroy()
	})

	b.instanceRegistry.Register(instance)
	return instance
}

func (b *bootstrap) snapshot_path() string {
	return path.Join(b.config.BaseDir, "db", "instances.json")
}

func (b *bootstrap) save_snapshot() {
	start := time.Now()

	instances := b.instanceRegistry.Instances()
	iSnaps := make([]map[string]interface{}, 0, len(instances))
	for _, i := range instances {
		switch i.State() {
		case starting.STATE_RUNNING, starting.STATE_CRASHED:
			iSnaps = append(iSnaps, i.Snapshot_attributes())
		}
	}

	snapshot := make(map[string]interface{})
	snapshot["time"] = start.Nanosecond()
	snapshot["instances"] = iSnaps

	bytes, err := goyaml.Marshal(snapshot)
	if err != nil {
		b.logger.Errorf("Erroring during snapshot marshalling: %s", err.Error())
		return

	}

	file, err := ioutil.TempFile(path.Join(b.config.BaseDir, "tmp"), "instances")
	if err != nil {
		b.logger.Errorf("Erroring during snapshot: %s", err.Error())
		return
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		b.logger.Errorf("Erroring during writing snapshot: %s", err.Error())
		return
	}
	file.Close()

	err = os.Rename(file.Name(), b.snapshot_path())
	if err != nil {
		b.logger.Errorf("Erroring during snapshot move: %s", err.Error())
		return
	}

	b.logger.Debugf("Saving snapshot took: %.3fs", (time.Now().Sub(start) / time.Second))
}

func (b *bootstrap) HandleHealthManagerStart(payload []byte) {
	b.sendHeartbeat(b.instanceRegistry.Instances())
}

func (b *bootstrap) HandleRouterStart(payload []byte) {
	var d map[string]interface{}
	if err := json.Unmarshal(payload, d); err != nil {
		b.logger.Errorf("Invalid Router Start payload: %s", payload)
		return
	}

	var interval time.Duration = 0
	if d != nil {
		interval = d["minimumRegisterIntervalInSeconds"].(time.Duration)
	}

	b.register_routes()

	if interval > 0 {
		if b.registrationTicker != nil {
			b.registrationTicker.Stop()
		}
		utils.Repeat(func() { b.register_routes() }, interval*time.Second)
	}

}

func (b *bootstrap) register_routes() {
	for _, i := range b.instanceRegistry.Instances() {
		if i.State() == starting.STATE_RUNNING || len(i.ApplicationUris()) > 0 {
			b.routerClient.RegisterInstance(i, nil)
		}
	}
}

func (b *bootstrap) HandleDeaStatus(payload []byte, replyTo cfmessage.ReplyTo) {
	response := protocol.NewDeaStatusResponse(b.component.UUID, b.localIp, b.directoryServer.Port,
		b.resource_manager.MemoryCapacity(),
		uint(b.resource_manager.ReservedMemory()), uint(b.resource_manager.UsedMemory()))
	if bytes, err := json.Marshal(response); err == nil {
		replyTo.Respond(bytes)
	} else {
		b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
	}
}

func (b *bootstrap) HandleDeaDirectedStart(payload []byte) {
	var d map[string]interface{}
	if err := json.Unmarshal(payload, d); err == nil {
		b.StartApp(d)
	} else {
		b.logger.Errorf("HandleDeaDirectedStart: marshal failed, %s", err.Error())
	}

}
func (b *bootstrap) HandleDeaStop(payload []byte) {
	var d map[string]interface{}
	if err := json.Unmarshal(payload, d); err == nil {
		b.instances_filtered_by_message(d, func(i *starting.Instance) {
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

func (b *bootstrap) HandleDeaUpdate(payload []byte) {
	var d map[string]interface{}
	if err := json.Unmarshal(payload, d); err != nil {
		b.logger.Errorf("HandleDeaUpdate: marshal failed, %s", err.Error())
	}
	app_id := d["droplet"].(string)
	uris := d["uris"].([]string)

	for _, i := range b.instanceRegistry.InstancesForApplication(app_id) {

		current_uris := i.ApplicationUris()

		b.logger.Debug("Mapping new URIs")
		b.logger.Debugf("New: %v Old: %v", uris, current_uris)

		new_uris := utils.Intersection(uris, current_uris)
		if len(new_uris) > 0 {
			b.routerClient.RegisterInstance(i, map[string]interface{}{"uris": new_uris})
		}

		obsolete_uris := utils.Intersection(current_uris, uris)
		if len(obsolete_uris) > 0 {
			b.routerClient.UnregisterInstance(i, map[string]interface{}{"uris": obsolete_uris})
		}

		i.SetApplicationUris(uris)
	}
}

func (b *bootstrap) HandleDeaFindDroplet(payload []byte, replyTo cfmessage.ReplyTo) {
	var d map[string]interface{}
	if err := json.Unmarshal(payload, d); err == nil {
		b.instances_filtered_by_message(d, func(i *starting.Instance) {
			response := protocol.NewFindDropletResponse(b.UUID(), b.localIp, i, b.directoryServer, b.directoryServerV2, d)
			if bytes, err := json.Marshal(response); err == nil {
				replyTo.Respond(bytes)
			} else {
				b.logger.Errorf("HandleDeaStatus: marshal failed, %s", err.Error())
			}
		})
	}
}

func (b *bootstrap) UUID() string {
	return b.component.UUID
}

func (b *bootstrap) instances_filtered_by_message(data map[string]interface{}, f func(*starting.Instance)) {
	app_id, exist := data["droplet"].(string)

	if !exist {
		b.logger.Warn("Filter message missing app_id")
		return
	}
	b.logger.Debug2f("Filter message for app_id: %s", app_id)

	instances := b.instanceRegistry.InstancesForApplication(app_id)
	if instances == nil {
		b.logger.Debug2f("No instances found for app_id: %s", app_id)
		return
	}

	// Optional search filters
	version := data["version"].(string)
	instance_ids := data["instances"].([]string)
	if ids, exists := data["instance_ids"].([]string); exists {
		instance_ids = append(instance_ids, ids...)
	}

	indices := data["indices"].([]int)
	states := data["states"].([]starting.State)

	for _, i := range instances {
		matched := true

		if version != "" {
			matched = matched && (i.ApplicationVersion() == version)
		}

		if instance_ids != nil {
			idMatch := false
			for _, id := range instance_ids {
				if id == i.Id() {
					idMatch = true
					break
				}
			}
			matched = matched && idMatch
		}
		if indices != nil {
			idxMatch := false
			for _, idx := range indices {
				if idx == i.Index() {
					idxMatch = true
					break
				}
			}
			matched = matched && idxMatch
		}

		if states != nil {
			statesMatch := false
			for _, state := range states {
				if state == i.State() {
					statesMatch = true
					break
				}
			}
			matched = matched && statesMatch
		}

		if matched {
			f(i)
		}
	}
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
	if d == nil {
		d := make(map[string]interface{})
		b.component.Varz.UniqueVarz = d
	}

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
