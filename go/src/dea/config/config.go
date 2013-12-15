package config

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"path"
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
	File   string `yaml:"file"`
	Syslog string `yaml:"syslog"`
	Level  string `yaml:"level"`
}

type LoggregatorConfig struct {
	Router       string `yaml:"router"`
	SharedSecret string `yaml:"shared_secret"`
}

type ResourcesConfig struct {
	MemoryMB               uint64  `yaml:"memory_mb"`
	MemoryOvercommitFactor float64 `yaml:"memory_overcommit_factor"`
	DiskMB                 uint64  `yaml:"disk_mb"`
	DiskOvercommitFactor   float64 `yaml:"disk_overcommit_factor"`
}

type PlatformConfig struct {
	Cache string `yaml:"cache"`
}

type StagingConfig struct {
	Enabled            bool              `yaml:"enabled"`
	Platform           PlatformConfig    `yaml:"platform_config"`
	Environment        map[string]string `yaml:"environment"`
	MemoryLimitMB      uint64            `yaml:"memory_limit_mb"`
	DiskLimitMB        uint64            `yaml:"disk_limit_mb"`
	MaxStagingDuration time.Duration     `yaml:"max_staging_duration"`
}

var defaultStagingConfig = StagingConfig{
	MaxStagingDuration: 900,
}

type DirServerConfig struct {
	Protocol         string `yaml:"protocol"`
	DeaPort          uint16 `yaml:"file_api_port"`
	V2Port           uint16 `"yaml:"v2_port"`
	StreamingTimeout uint32 `yaml:"streaming_timeout"`
}

type NatsConfig struct {
	Host string
	Port uint16
	User string
	Pass string
}

type StatusConfig struct {
	Port     uint16 `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

var defaultNatsConfig = NatsConfig{
	Host: "localhost",
	Port: 4222,
	User: "",
	Pass: "",
}

type IntervalConfig struct {
	Heartbeat time.Duration `yaml:"heartbeat"`
	Advertise time.Duration `yaml:"advertise"`
}

type Config struct {
	BaseDir                       string                 `yaml:"base_dir"`
	BuildpackDir                  string                 `yaml:"buildpack_dir"`
	Logging                       LoggingConfig          `yaml:"logging"`
	Loggregator                   LoggregatorConfig      `yaml:"loggregator"`
	Resources                     ResourcesConfig        `yaml:"resources"`
	NatsServers                   []string               `yaml:"nats_servers"`
	PidFile                       string                 `yaml:"pid_filename"`
	WardenSocket                  string                 `yaml:"warden_socket"`
	EvacuationBailOut             time.Duration          `yaml:"evacuation_bail_out_time_in_seconds"`
	MaximumHealthCheckTimeout     time.Duration          `yaml:"maximum_health_check_timeout"`
	Index                         uint                   `yaml:"index"`
	Staging                       StagingConfig          `yaml:"staging"`
	Stacks                        []string               `yaml:"stacks"`
	Intervals                     IntervalConfig         `yaml:"intervals"`
	Status                        StatusConfig           `yaml:"status"`
	CrashesPath                   string                 `yaml:"crashes_path"`
	CrashLifetime                 time.Duration          `yaml:"crash_lifetime_secs"`
	BindMounts                    []map[string]string    `yaml:"bind_mounts"`
	CrashBlockUsageRatioThreshold float64                `yaml:"crash_block_usage_ratio_threshold"`
	CrashInodeUsageRatioThreshold float64                `yaml:"crash_inode_usage_ratio_threshold"`
	Domain                        string                 `yaml:"domain"`
	DirectoryServer               DirServerConfig        `yaml:"directory_server"`
	DeaRuby                       string                 `yaml:"dea_ruby"`
	Hooks                         map[string]string      `yaml:"hooks"`
	PlacementProperties           map[string]interface{} `yaml:"placement_properties"`
}

type ConfigLoader func(c *Config) error

func NewConfig(loader ConfigLoader) (Config, error) {
	c := Config{
		Status:                        StatusConfig{},
		Resources:                     ResourcesConfig{},
		CrashLifetime:                 60 * 60,
		EvacuationBailOut:             10 * 60,
		BindMounts:                    []map[string]string{},
		CrashBlockUsageRatioThreshold: 0.8,
		CrashInodeUsageRatioThreshold: 0.8,
		MaximumHealthCheckTimeout:     60,
		PlacementProperties:           map[string]interface{}{},
		Staging: StagingConfig{
			MemoryLimitMB: 1024,
			DiskLimitMB:   2 * 1024,
		},
	}

	var err error
	if loader != nil {
		err = loader(&c)
	}

	if err == nil {
		err = finalize(&c)
	}

	return c, err
}

func LoadConfig(configPath string) (Config, error) {
	return NewConfig(func(c *Config) error {
		configBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		if err := goyaml.Unmarshal(configBytes, c); err != nil {
			return err
		}

		return nil
	})
}

func finalize(config *Config) error {
	if config.CrashesPath == "" {
		config.CrashesPath = path.Join(config.BaseDir, "crashes")
	}

	// fix up durations
	config.Staging.MaxStagingDuration *= time.Second
	config.EvacuationBailOut *= time.Second
	config.MaximumHealthCheckTimeout *= time.Second
	config.CrashLifetime *= time.Second
	config.Intervals.Advertise *= time.Second
	config.Intervals.Heartbeat *= time.Second

	return nil
}
