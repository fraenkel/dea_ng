package staging

import (
	cfg "dea/config"
	dstaging "dea/staging"
	"strings"
	"time"
)

type MockStagingTask struct {
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

func (m *MockStagingTask) Id() string {
	return m.StagingMessage().Task_id()
}

func (m *MockStagingTask) StagingMessage() dstaging.StagingMessage {
	return m.StagingMsg
}

func (m *MockStagingTask) StagingConfig() cfg.StagingConfig {
	return m.StagingCfg
}

func (m *MockStagingTask) MemoryLimit() cfg.Memory {
	return cfg.Memory(m.StagingCfg.MemoryLimitMB) * cfg.Mebi
}

func (m *MockStagingTask) DiskLimit() cfg.Disk {
	return cfg.Disk(m.StagingCfg.DiskLimitMB) * cfg.MB
}

func (m *MockStagingTask) Start() error {
	m.Started = true
	return nil
}

func (m *MockStagingTask) Stop() {
	m.Stopped = true
}

func (m *MockStagingTask) StreamingLogUrl(maker dstaging.StagingTaskUrlMaker) string {
	return m.Streaming_log_url
}
func (m *MockStagingTask) DetectedBuildpack() string {
	return ""
}
func (m *MockStagingTask) DropletSHA1() string {
	return m.Droplet_Sha1
}
func (m *MockStagingTask) Path_in_container(pathSuffix string) string {
	cPath := m.ContainerPath
	if cPath == "" {
		return ""
	}

	// Do not use path.Join since the result is Cleaned
	return strings.Join([]string{cPath, "tmp", "rootfs", pathSuffix}, "/")
}

func (m *MockStagingTask) SetAfter_setup_callback(callback dstaging.Callback) {
	m.SetupCallback = callback
}
func (m *MockStagingTask) SetAfter_complete_callback(callback dstaging.Callback) {
	m.CompleteCallback = callback
}
func (m *MockStagingTask) SetAfter_upload_callback(callback dstaging.Callback) {
	m.UploadCallback = callback
}
func (m *MockStagingTask) SetAfter_stop_callback(callback dstaging.Callback) {
	m.StopCallback = callback
}
func (m *MockStagingTask) StagingTimeout() time.Duration {
	return m.StagingCfg.MaxStagingDuration
}
