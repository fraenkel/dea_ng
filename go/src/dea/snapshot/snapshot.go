package snapshot

import (
	"dea"
	"dea/utils"
	steno "github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path"
	"time"
)

type Snap struct {
	Time          int64
	Instances     []map[string]interface{}
	Staging_tasks []map[string]interface{}
}

type snapshot struct {
	instanceManager     dea.InstanceManager
	instanceRegistry    dea.InstanceRegistry
	stagingTaskRegistry dea.StagingTaskRegistry
	snapPath            string
	tmpPath             string
	logger              *steno.Logger
}

func NewSnapshot(ir dea.InstanceRegistry, str dea.StagingTaskRegistry, im dea.InstanceManager, baseDir string) dea.Snapshot {
	return &snapshot{
		instanceRegistry:    ir,
		stagingTaskRegistry: str,
		instanceManager:     im,
		snapPath:            path.Join(baseDir, "db", "instances.json"),
		tmpPath:             path.Join(baseDir, "tmp"),
		logger:              utils.Logger("Snapshot", nil),
	}
}

func (s *snapshot) Path() string {
	return s.snapPath
}

func (s *snapshot) Load() error {
	snapshot_path := s.Path()
	if !utils.File_Exists(snapshot_path) {
		return nil
	}

	start := time.Now()
	snap := Snap{}
	err := utils.Yaml_Load(snapshot_path, &snap)
	if err != nil {
		return err
	}

	for _, attrs := range snap.Instances {
		instance_state := dea.State(attrs["state"].(string))
		println("state", instance_state)
		delete(attrs, "state")
		instance := s.instanceManager.CreateInstance(attrs)
		if instance == nil {
			continue
		}

		// Enter instance state via "RESUMING" to trigger the right transitions
		instance.SetState(dea.STATE_RESUMING)
		instance.SetState(instance_state)
	}

	s.logger.Debugf("Loading snapshot took: %.3fs", time.Now().Sub(start).Seconds())
	return nil
}

func (s *snapshot) Save() {
	start := time.Now()

	instances := s.instanceRegistry.Instances()
	iSnaps := make([]map[string]interface{}, 0, len(instances))
	for _, i := range instances {
		switch i.State() {
		case dea.STATE_RUNNING, dea.STATE_CRASHED:
			iSnaps = append(iSnaps, i.Snapshot_attributes())
		}
	}

	stagings := s.stagingTaskRegistry.Tasks()
	sSnaps := make([]map[string]interface{}, 0, len(stagings))
	for _, s := range stagings {
		sSnaps = append(sSnaps, s.StagingMessage().AsMap())

	}

	snap := Snap{}
	snap.Time = start.UnixNano()
	snap.Instances = iSnaps
	snap.Staging_tasks = sSnaps

	bytes, err := goyaml.Marshal(snap)
	if err != nil {
		s.logger.Errorf("Erroring during snapshot marshalling: %s", err.Error())
		return

	}

	file, err := ioutil.TempFile(s.tmpPath, "snapshot")
	if err != nil {
		s.logger.Errorf("Erroring during snapshot: %s", err.Error())
		return
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		s.logger.Errorf("Erroring during writing snapshot: %s", err.Error())
		return
	}
	file.Close()

	err = os.Rename(file.Name(), s.Path())
	if err != nil {
		s.logger.Errorf("Erroring during snapshot move: %s", err.Error())
		return
	}

	s.logger.Debugf("Saving snapshot took: %.3fs", (time.Now().Sub(start) / time.Second))
}
