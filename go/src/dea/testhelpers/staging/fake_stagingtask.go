package staging

import (
	"dea"
	cfg "dea/config"
	steno "github.com/cloudfoundry/gosteno"
	"strings"
	"time"
)

func FakeStagingTaskCreator(config *cfg.Config, msg dea.StagingMessage, bps []dea.StagingBuildpack, dr dea.DropletRegistry, logger *steno.Logger) dea.StagingTask {
	return &FakeStagingTask{StagingMsg: msg,
		StagingCfg: config.Staging,
	}
}

type FakeStagingTask struct {
	StagingMsg        dea.StagingMessage
	StagingCfg        cfg.StagingConfig
	Streaming_log_url string
	Droplet_Sha1      string
	ContainerPath     string
	Started           bool
	Stopped           bool

	InvokeSetup   bool
	SetupCallback dea.Callback
	SetupError    error

	InvokeComplete   bool
	CompleteCallback dea.Callback
	CompleteError    error

	InvokeStop   bool
	StopCallback dea.Callback
	StopError    error
}

func (fst *FakeStagingTask) Id() string {
	return fst.StagingMessage().Task_id()
}

func (fst *FakeStagingTask) StagingMessage() dea.StagingMessage {
	return fst.StagingMsg
}

func (fst *FakeStagingTask) StagingConfig() cfg.StagingConfig {
	return fst.StagingCfg
}

func (fst *FakeStagingTask) MemoryLimit() cfg.Memory {
	return cfg.Memory(fst.StagingCfg.MemoryLimitMB) * cfg.Mebi
}

func (fst *FakeStagingTask) DiskLimit() cfg.Disk {
	return cfg.Disk(fst.StagingCfg.DiskLimitMB) * cfg.MB
}

func (fst *FakeStagingTask) Start() {
	fst.Started = true
}

func (fst *FakeStagingTask) Stop(callback dea.Callback) {
	fst.Stopped = true
	if callback != nil {
		go callback(nil)
	}
}

func (fst *FakeStagingTask) StreamingLogUrl(maker dea.StagingTaskUrlMaker) string {
	return fst.Streaming_log_url
}
func (fst *FakeStagingTask) DetectedBuildpack() string {
	return ""
}
func (fst *FakeStagingTask) DropletSHA1() string {
	return fst.Droplet_Sha1
}
func (fst *FakeStagingTask) Path_in_container(pathSuffix string) string {
	cPath := fst.ContainerPath
	if cPath == "" {
		return ""
	}

	// Do not use path.Join since the result is Cleaned
	return strings.Join([]string{cPath, "tmp", "rootfs", pathSuffix}, "/")
}

func (fst *FakeStagingTask) SetAfter_setup_callback(callback dea.Callback) {
	fst.SetupCallback = callback

	if fst.InvokeSetup {
		callback(fst.SetupError)
	}
}
func (fst *FakeStagingTask) SetAfter_complete_callback(callback dea.Callback) {
	fst.CompleteCallback = callback
	if fst.InvokeComplete {
		callback(fst.CompleteError)
	}
}
func (fst *FakeStagingTask) SetAfter_stop_callback(callback dea.Callback) {
	fst.StopCallback = callback
	if fst.InvokeStop {
		callback(fst.StopError)
	}
}
func (fst *FakeStagingTask) StagingTimeout() time.Duration {
	return fst.StagingCfg.MaxStagingDuration
}
