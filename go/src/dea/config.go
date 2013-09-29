package dea

import (
	"io/ioutil"
	"launchpad.net/goyaml"
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

type Config struct {
	BaseDir                       string            "base_dir"
	Logging                       LoggingConfig     "logging"
	Loggregator                   LoggregatorConfig "loggregator"
	Resources                     ResourcesConfig   "resources"
	NatsURI                       string            "nats_uri"
	PIDFilename                   string            "pid_filename"
	WardenSocket                  string            "warden_socket"
	EvacuationDelay               int               "evacuation_delay_secs"
	Index                         int               "index"
	Staging                       StagingConfig     "staging"
	Stacks                        []string
	Intervals                     map[int]int "intervals"
	Status                        map[int]int "status"
	CrashLifetime                 int         "crash_lifetime_secs"
	BindMounts                    []int       "bind_mounts"
	CrashBlockUsageRatioThreshold float32     "crash_block_usage_ratio_threshold"
	CrashInodeUsageRatioThreshold float32     "crash_inode_usage_ratio_threshold"
}

func ConfigFromFile(path string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := Config{Intervals: make(map[int]int),
		Status:                        make(map[int]int),
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

	return &config, nil
}
