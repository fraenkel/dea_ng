package staging

import (
	cfg "dea/config"
	dstaging "dea/staging"
	"strings"
	"time"
)

type FakeStagingTask struct {
	StagingMsg        dstaging.StagingMessage
	StagingCfg        cfg.StagingConfig
	Streaming_log_url string
	Droplet_Sha1      string
	ContainerPath     string
	Started           bool
	Stopped           bool

	InvokeSetup   bool
	SetupCallback dstaging.Callback
	SetupError    error

	InvokeComplete   bool
	CompleteCallback dstaging.Callback
	CompleteError    error

	InvokeStop   bool
	StopCallback dstaging.Callback
	StopError    error
}

func (fst *FakeStagingTask) Id() string {
	return fst.StagingMessage().Task_id()
}

func (fst *FakeStagingTask) StagingMessage() dstaging.StagingMessage {
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

func (fst *FakeStagingTask) Start() error {
	fst.Started = true
	return nil
}

func (fst *FakeStagingTask) Stop() {
	fst.Stopped = true
}

func (fst *FakeStagingTask) StreamingLogUrl(maker dstaging.StagingTaskUrlMaker) string {
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

func (fst *FakeStagingTask) SetAfter_setup_callback(callback dstaging.Callback) {
	fst.SetupCallback = callback

	if fst.InvokeSetup {
		callback(fst.SetupError)
	}
}
func (fst *FakeStagingTask) SetAfter_complete_callback(callback dstaging.Callback) {
	fst.CompleteCallback = callback
	if fst.InvokeComplete {
		callback(fst.CompleteError)
	}
}
func (fst *FakeStagingTask) SetAfter_stop_callback(callback dstaging.Callback) {
	fst.StopCallback = callback
	if fst.InvokeStop {
		callback(fst.StopError)
	}
}
func (fst *FakeStagingTask) StagingTimeout() time.Duration {
	return fst.StagingCfg.MaxStagingDuration
}
