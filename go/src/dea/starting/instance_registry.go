package starting

import (
	"dea/config"
	"dea/utils"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

const (
	DEFAULT_CRASH_LIFETIME_SECS  = 60 * 60
	CRASHES_REAPER_INTERVAL_SECS = 10
)

type InstanceRegistry struct {
	instances                     map[string]*Instance
	instancesByAppId              map[string]map[string]*Instance
	crashLifetime                 time.Duration
	crashesPath                   string
	crashBlockUsageRatioThreshold float64
	crashInodeUsageRatioThreshold float64
	sync.Mutex
	emitter emitter.Emitter
	logger  *steno.Logger
}

type instances []*Instance

type byTimestamp struct {
	instances
}

func (s byTimestamp) Len() int {
	return len(s.instances)
}
func (s byTimestamp) Less(i, j int) bool {
	return s.instances[i].StateTimestamp().Before(s.instances[j].StateTimestamp())
}

func (s byTimestamp) Swap(i, j int) {
	s.instances[i], s.instances[j] = s.instances[j], s.instances[i]
}

func NewInstanceRegistry(config *config.Config) *InstanceRegistry {
	registry := &InstanceRegistry{
		crashesPath:                   config.CrashesPath,
		crashBlockUsageRatioThreshold: config.CrashBlockUsageRatioThreshold,
		crashInodeUsageRatioThreshold: config.CrashInodeUsageRatioThreshold,
		logger: steno.NewLogger("InstanceRegistry"),
	}
	crashLifetime := config.CrashLifetime
	if crashLifetime == 0 {
		crashLifetime = DEFAULT_CRASH_LIFETIME_SECS
	}
	registry.crashLifetime = time.Duration(crashLifetime)

	return registry
}

func (r *InstanceRegistry) Instances() []*Instance {
	r.Lock()
	defer r.Unlock()

	instances := make([]*Instance, 0, len(r.instances))
	for _, instance := range r.instances {
		instances = append(instances, instance)
	}
	return instances
}

func (r *InstanceRegistry) InstancesForApplication(app_id string) map[string]*Instance {
	r.Lock()
	defer r.Unlock()

	return r.instancesByAppId[app_id]
}

func (r *InstanceRegistry) Register(instance *Instance) {
	applicationId := instance.ApplicationId()
	r.emitter.Emit(applicationId, "Registering instance")
	r.logger.Debug2f("Registering instance %s", instance.Id())

	r.Lock()
	defer r.Unlock()

	r.instances[instance.Id()] = instance

	instances := r.instancesByAppId[applicationId]
	if instances == nil {
		instances = make(map[string]*Instance, 1)
		r.instancesByAppId[applicationId] = instances
	}
	instances[instance.Id()] = instance
}

func (r *InstanceRegistry) Unregister(instance *Instance) {
	applicationId := instance.ApplicationId()

	r.emitter.Emit(applicationId, "Removing instance")
	r.logger.Debug2f("Removing instance %s", instance.Id())

	r.Lock()
	defer r.Unlock()

	delete(r.instances, instance.Id())
	instances := r.instancesByAppId[applicationId]
	if instances != nil {
		delete(instances, instance.Id())
		if len(instances) == 0 {
			delete(r.instancesByAppId, applicationId)
		}
	}
}

func (r *InstanceRegistry) lookupInstance(instanceId string) *Instance {
	r.Lock()
	defer r.Unlock()
	return r.instances[instanceId]
}

func (r *InstanceRegistry) StartReapers() {
	utils.Repeat(func() {
		r.reapOrphanedCrashes()
		r.reapCrashes()
		r.reapCrashesUnderDiskPressure()
	}, CRASHES_REAPER_INTERVAL_SECS)
}

func (r *InstanceRegistry) reapOrphanedCrashes() {
	r.logger.Debug2("Reaping orphaned crashes")

	crashes := make([]string, 0)
	filepath.Walk(r.crashesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == r.crashesPath {
			return nil
		}
		if info.IsDir() {
			crashes = append(crashes, filepath.Base(path))
		}
		return filepath.SkipDir
	})

	for _, instanceId := range crashes {
		instance := r.lookupInstance(instanceId)
		if instance != nil {
			r.reapCrash(instanceId, "orphaned", nil)
		}
	}
}

func (r *InstanceRegistry) reapCrashes() {
	r.logger.Debug2("Reaping crashes")

	crashesByApp := make(map[string][]*Instance)
	for _, instance := range r.Instances() {
		if instance.State() == STATE_CRASHED {
			crashedApps := crashesByApp[instance.ApplicationId()]
			if crashedApps == nil {
				crashedApps = make([]*Instance, 0, 1)
			}
			crashesByApp[instance.ApplicationId()] = append(crashedApps, instance)
		}
	}

	now := time.Now()
	for _, crashedInstances := range crashesByApp {
		sort.Sort(byTimestamp{crashedInstances})
		// Remove if not most recent, or too old
		for idx, instance := range crashedInstances {
			if idx > 0 || now.Sub(instance.StateTimestamp()) > r.crashLifetime {
				r.reapCrash(instance.Id(), "stale", nil)
			}
		}
	}
}

func (r *InstanceRegistry) reapCrashesUnderDiskPressure() {
	r.logger.Debug2("Reaping crashes under disk pressure")

	if r.hasDiskPressure() {
		crashed := make([]*Instance, 0)
		for _, instance := range r.Instances() {
			if instance.State() == STATE_CRASHED {
				crashed = append(crashed, instance)
			}
		}
		sort.Sort(byTimestamp{crashed})

		// Remove oldest crash
		if len(crashed) > 0 {
			r.reapCrash(crashed[0].Id(), "disk pressure", func() {
				// Continue reaping crashes when done
				r.reapCrashesUnderDiskPressure()
			})
		}
	}
}

func (r *InstanceRegistry) hasDiskPressure() bool {
	result := false
	stat := syscall.Statfs_t{}
	err := syscall.Statfs(r.crashesPath, &stat)
	if err != nil {
		r.logger.Error(err.Error())
		return false
	}

	block_usage_ratio := float64(stat.Blocks-stat.Bfree) / float64(stat.Blocks)
	inode_usage_ratio := float64(stat.Files-stat.Ffree) / float64(stat.Files)

	result = result || block_usage_ratio > r.crashBlockUsageRatioThreshold
	result = result || inode_usage_ratio > r.crashInodeUsageRatioThreshold

	if result {
		r.logger.Debugf("Disk usage (block/inode): %.3f/%.3f", block_usage_ratio, inode_usage_ratio)
	}

	return result
}

func (r *InstanceRegistry) reapCrash(instanceId string, reason string, callback func()) {
	instance := r.lookupInstance(instanceId)

	data := map[string]string{
		"instance_id": instanceId,
		"reason":      reason,
	}

	if instance != nil {
		applicationId := instance.ApplicationId()
		data["application_id"] = applicationId
		data["application_version"] = instance.ApplicationVersion()
		data["application_name"] = instance.ApplicationName()

		r.emitter.Emit(applicationId, "Removing crash for app with id "+applicationId)
	}

	message := "Removing crash " + instanceId

	if instance != nil {
		r.Unregister(instance)
	}

	r.logger.Debugf("%s:  %v", message, data)

	t := time.Now()

	r.destroyCrashArtifacts(instanceId)
	r.logger.Debugf("%s : took %.3fs %v", message, time.Now().Sub(t).Seconds(), data)
	if callback != nil {
		callback()
	}
}

func (r *InstanceRegistry) destroyCrashArtifacts(instanceId string) {
	crashPath := filepath.Join(r.crashesPath, instanceId)
	r.logger.Debug2("Removing path " + crashPath)
	if err := os.RemoveAll(crashPath); err != nil {
		r.logger.Error(err.Error())
	}
}

func (r *InstanceRegistry) AppIdToCount() map[string]int {
	appCounts := make(map[string]int)

	r.Lock()
	defer r.Unlock()

	for k, v := range r.instancesByAppId {
		appCounts[k] = len(v)
	}
	return appCounts
}

func (r *InstanceRegistry) ReservedMemory() config.Memory {
	var sum config.Memory = 0
	for _, i := range r.Instances() {
		if i.IsAlive() {
			sum = sum + i.MemoryLimit()
		}
	}
	return sum
}

func (r *InstanceRegistry) UsedMemory() config.Memory {
	var sum config.Memory = 0
	for _, i := range r.Instances() {
		sum = sum + i.UsedMemory()
	}
	return sum
}

func (r *InstanceRegistry) ReservedDisk() config.Disk {
	var sum config.Disk = 0
	for _, i := range r.Instances() {
		if i.IsAlive() {
			sum = sum + i.DiskLimit()
		}
	}
	return sum
}

func (r *InstanceRegistry) ToHash() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	for _, i := range r.Instances() {
		apps := result[i.ApplicationId()]
		if apps == nil {
			apps = make(map[string]interface{})
			result[i.ApplicationId()] = apps
		}
		apps[i.Id()] = i.attributes
	}
	return result
}
