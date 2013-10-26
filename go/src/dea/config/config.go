package config

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Disk uint64

/* Multiples of the unit byte, generally for computer storage. */
const (
	KB = Disk(1e3)  // kilobyte
	MB = Disk(1e6)  // megabyte
	GB = Disk(1e9)  // gigabyte
	TB = Disk(1e12) // terabyte
	PB = Disk(1e15) // petabyte
	EB = Disk(1e18) // exabyte
)

type Memory uint64

/* Multiples of the unit byte, generally for computer memory. */
const (
	_ = Memory(1 << (10 * iota))
	Kibi
	Mebi
	Gibi
	Tebi
	Pebi
	Exbi
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
	MemoryMB               uint64  "memory_mb"
	MemoryOvercommitFactor float64 "memory_overcommit_factor"
	DiskMB                 uint64  "disk_mb"
	DiskOvercommitFactor   float64 "disk_overcommit_factor"
}

type PlatformConfig struct {
	Cache string "cache"
}

type StagingConfig struct {
	Enabled            bool              "enabled"
	Platform           PlatformConfig    "platform_config"
	Environment        map[string]string "environment"
	MemoryLimitMB      uint64            "memory_limit_mb"
	DiskLimitMB        uint64            "disk_limit_mb"
	MaxStagingDuration time.Duration     "max_staging_duration"
}

var defaultStagingConfig = StagingConfig{
	MaxStagingDuration: 900 * time.Second,
}

type DirServerConfig struct {
	Protocol         string "protocol"
	DeaPort          uint16 "file_api_port"
	V1Port           uint16 "v1_port"
	V2Port           uint16 "v2_port"
	StreamingTimeout uint32 "streaming_timeout"
}

type NatsConfig struct {
	Host string "host"
	Port uint16 "port"
	User string "user"
	Pass string "pass"
}

type StatusConfig struct {
	Port     uint16 "port"
	User     string "user"
	Password string "password"
}

var defaultNatsConfig = NatsConfig{
	Host: "localhost",
	Port: 4222,
	User: "",
	Pass: "",
}

type Config struct {
	BaseDir                       string                   "base_dir"
	BuildpackDir                  string                   "buildpack_dir"
	Logging                       LoggingConfig            "logging"
	Loggregator                   LoggregatorConfig        "loggregator"
	Resources                     ResourcesConfig          "resources"
	natsUri                       string                   "nats_uri"
	NatsConfig                    NatsConfig               "nats"
	PidFile                       string                   "pid_filename"
	WardenSocket                  string                   "warden_socket"
	EvacuationDelay               time.Duration            "evacuation_delay_secs"
	Index                         uint                     "index"
	Staging                       StagingConfig            "staging"
	Stacks                        []string                 "stacks"
	Intervals                     map[string]time.Duration "intervals"
	Status                        StatusConfig             "status"
	CrashesPath                   string                   "crashes_path"
	CrashLifetime                 time.Duration            "crash_lifetime_secs"
	BindMounts                    []map[string]string      "bind_mounts"
	CrashBlockUsageRatioThreshold float64                  "crash_block_usage_ratio_threshold"
	CrashInodeUsageRatioThreshold float64                  "crash_inode_usage_ratio_threshold"
	Domain                        string                   "domain"
	DirectoryServer               DirServerConfig          "directory_server"
	DeaRuby                       string                   "dea_ruby"
	Hooks                         map[string]string        "hooks"
}

func ConfigFromFile(configPath string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	config := Config{
		Intervals:                     make(map[string]time.Duration),
		Status:                        StatusConfig{},
		NatsConfig:                    defaultNatsConfig,
		Resources:                     ResourcesConfig{},
		CrashLifetime:                 60 * 60,
		EvacuationDelay:               30,
		BindMounts:                    []map[string]string{},
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

	if config.CrashesPath == "" {
		config.CrashesPath = path.Join(config.BaseDir, "crashes")
	}

	return &config, nil
}
