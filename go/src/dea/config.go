package dea

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/url"
	"strconv"
	"strings"
)

type LoggingConfig struct {
	File   string "file"
	Syslog string "syslog"
	Level  string "level"
}

type LoggregatorConfig struct {
	Router string "router"
}

type ResourcesConfig struct {
	MemoryMb               int     "memory_mb"
	MemoryOvercommitFactor float32 "memory_overcommit_factor"
	DiskMb                 int     "disk_mb"
	DiskOvercommitFactor   float32 "disk_overcommit_factor"
}

type PlatformConfig struct {
	Cache string "cache"
}

type StagingConfig struct {
	Enabled            bool              "enabled"
	Platform           PlatformConfig    "platform_config"
	Environment        map[string]string "environment"
	MemoryLimitMB      int               "memory_limit_mb"
	DiskLimitMB        int               "disk_limit_mb"
	MaxStagingDuration int               "max_staging_duration"
}

type DirServerConfig struct {
	DeaPort          uint16 "file_api_port"
	DirServerV1Port  uint16 "v1_port"
	DirServerV2Port  uint16 "v2_port"
	StreamingTimeout uint32 "streaming_timeout"
}

type NatsConfig struct {
	Host string "host"
	Port uint16 "port"
	User string "user"
	Pass string "pass"
}

var defaultNatsConfig = NatsConfig{
	Host: "localhost",
	Port: 4222,
	User: "",
	Pass: "",
}

type Config struct {
	BaseDir                       string            "base_dir"
	Logging                       LoggingConfig     "logging"
	Loggregator                   LoggregatorConfig "loggregator"
	Resources                     ResourcesConfig   "resources"
	natsUri                       string            "nats_uri"
	NatsConfig                    NatsConfig        "nats"
	PidFile                       string            "pid_filename"
	WardenSocket                  string            "warden_socket"
	EvacuationDelay               int               "evacuation_delay_secs"
	Index                         int               "index"
	Staging                       StagingConfig     "staging"
	Stacks                        []string
	Intervals                     map[string]int  "intervals"
	Status                        map[string]int  "status"
	CrashLifetime                 int             "crash_lifetime_secs"
	BindMounts                    []int           "bind_mounts"
	CrashBlockUsageRatioThreshold float32         "crash_block_usage_ratio_threshold"
	CrashInodeUsageRatioThreshold float32         "crash_inode_usage_ratio_threshold"
	DirectoryServer               DirServerConfig "directory_server"
}

func ConfigFromFile(path string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := Config{
		Intervals:                     make(map[string]int),
		Status:                        make(map[string]int),
		NatsConfig:                    defaultNatsConfig,
		Resources:                     ResourcesConfig{},
		CrashLifetime:                 60 * 60,
		EvacuationDelay:               30,
		BindMounts:                    []int{},
		CrashBlockUsageRatioThreshold: 0.8,
		CrashInodeUsageRatioThreshold: 0.8,
	}

	if err := goyaml.Unmarshal(configBytes, &config); err != nil {
		return nil, err
	}

	// Convert NatsUri -> NatsConfig
	if config.natsUri != "" {
		natsURL, err := url.Parse(config.natsUri)
		if err != nil {
			return nil, err
		}

		hostInfo := strings.SplitN(natsURL.Host, ":", 2)
		port, err := strconv.ParseUint(hostInfo[1], 0, 16)
		if err != nil {
			return nil, err
		}
		config.NatsConfig.Host = hostInfo[0]
		config.NatsConfig.Port = uint16(port)

		if natsURL.User != nil {
			config.NatsConfig.User = natsURL.User.Username()
			config.NatsConfig.Pass, _ = natsURL.User.Password()
		}
	}

	return &config, nil
}
