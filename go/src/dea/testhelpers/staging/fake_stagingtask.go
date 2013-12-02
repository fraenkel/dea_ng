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
	SetupCallback     dstaging.Callback
	CompleteCallback  dstaging.Callback
	UploadCallback    dstaging.Callback
	StopCallback      dstaging.Callback
}

func (m *FakeStagingTask) Id() string {
	return m.StagingMessage().Task_id()
}

func (m *FakeStagingTask) StagingMessage() dstaging.StagingMessage {
	return m.StagingMsg
}

func (m *FakeStagingTask) StagingConfig() cfg.StagingConfig {
	return m.StagingCfg
}

func (m *FakeStagingTask) MemoryLimit() cfg.Memory {
	return cfg.Memory(m.StagingCfg.MemoryLimitMB) * cfg.Mebi
}

func (m *FakeStagingTask) DiskLimit() cfg.Disk {
	return cfg.Disk(m.StagingCfg.DiskLimitMB) * cfg.MB
}

func (m *FakeStagingTask) Start() error {
	m.Started = true
	return nil
}

func (m *FakeStagingTask) Stop() {
	m.Stopped = true
}

func (m *FakeStagingTask) StreamingLogUrl(maker dstaging.StagingTaskUrlMaker) string {
	return m.Streaming_log_url
}
func (m *FakeStagingTask) DetectedBuildpack() string {
	return ""
}
func (m *FakeStagingTask) DropletSHA1() string {
	return m.Droplet_Sha1
}
func (m *FakeStagingTask) Path_in_container(pathSuffix string) string {
	cPath := m.ContainerPath
	if cPath == "" {
		return ""
	}

	// Do not use path.Join since the result is Cleaned
	return strings.Join([]string{cPath, "tmp", "rootfs", pathSuffix}, "/")
}

func (m *FakeStagingTask) SetAfter_setup_callback(callback dstaging.Callback) {
	m.SetupCallback = callback
}
func (m *FakeStagingTask) SetAfter_complete_callback(callback dstaging.Callback) {
	m.CompleteCallback = callback
}
func (m *FakeStagingTask) SetAfter_upload_callback(callback dstaging.Callback) {
	m.UploadCallback = callback
}
func (m *FakeStagingTask) SetAfter_stop_callback(callback dstaging.Callback) {
	m.StopCallback = callback
}
func (m *FakeStagingTask) StagingTimeout() time.Duration {
	return m.StagingCfg.MaxStagingDuration
}
