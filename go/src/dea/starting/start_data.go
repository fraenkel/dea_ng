package starting

import (
	"reflect"
	"strings"
)

type LimitsData struct {
	memMb  uint64
	diskMb uint64
	fds    uint64
}

type ServiceData struct {
	name             string
	label            string
	credentials      map[string]interface{}
	tags             []string
	syslog_drain_url string

	options map[string]interface{}

	provider string
	vendor   string
	plan     string

	version string
}

type StartData struct {
	cc_partition string

	instance_index int

	application_id      string
	application_version string
	application_name    string
	application_uris    []string

	prod bool

	droplet_sha1 string
	droplet_uri  string

	start_command string

	limits LimitsData

	environment map[string]string
	services    []ServiceData
}

func NewStartData(m map[string]interface{}) StartData {
	sd := StartData{}
	sd.UnmarshalMap(m)

	return sd
}

func (s StartData) Index() int {
	return s.instance_index
}

func (s StartData) Limits() map[string]uint64 {
	limit := make(map[string]uint64)
	s.limits.MarshalMap(limit)
	return limit
}

func (s StartData) MemoryLimit() uint64 {
	return uint64(s.limits.memMb)
}

func (s StartData) Name() string {
	return s.application_name
}

func (s StartData) Version() string {
	return s.application_version
}
func (s StartData) Uris() []string {
	return s.application_uris
}

func (s StartData) Services() []map[string]interface{} {
	services := make([]map[string]interface{}, len(s.services))
	for i, s := range s.services {
		service := make(map[string]interface{})
		s.MarshalMap(service)
		services[i] = service
	}
	return services
}

func (s StartData) Environment() map[string]string {
	return s.environment
}

func (s *StartData) UnmarshalMap(m map[string]interface{}) error {
	s.cc_partition = m["cc_partition"].(string)

	s.instance_index = getInt(m, "instance_index")

	s.application_version = getString(m, "application_version")
	if s.application_version == "" {
		s.application_version = getString(m, "version")
	}
	s.application_name = getString(m, "application_name")
	if s.application_name == "" {
		s.application_name = getString(m, "name")
	}
	s.application_uris = m["application_uris"].([]string)
	if s.application_uris == nil {
		s.application_uris = m["uris"].([]string)

		if s.application_uris == nil {
			s.application_uris = []string{}
		}
	}

	s.prod = getBool(m, "prod")

	s.application_id = getString(m, "application_id")
	if s.application_id == "" {
		s.application_id = getString(m, "droplet")
	}

	s.droplet_sha1 = getString(m, "droplet_sha1")
	if s.droplet_sha1 == "" {
		s.droplet_sha1 = getString(m, "sha1")
	}

	s.droplet_uri = getString(m, "droplet_uri")
	if s.droplet_uri == "" {
		s.droplet_uri = getString(m, "executableUri")
	}

	s.start_command = getString(m, "start_command")

	if env, exist := m["environment"]; exist {
		s.environment = env.(map[string]string)
	} else {
		env := make(map[string]string)
		s.environment = env
		for _, e := range m["env"].([]string) {
			pair := strings.SplitN(string(e), "=", 2)
			val := ""
			if len(pair) == 2 {
				val = pair[1]
			}
			env[pair[0]] = val
		}
	}

	s.limits.UnmarshalMap(m["limits"].(map[string]interface{}))

	serviceData := m["services"].([]map[string]interface{})
	s.services = make([]ServiceData, len(serviceData))
	for i := 0; i < len(s.services); i++ {
		s.services[i].UnmarshalMap(serviceData[i])
	}

	return nil
}

func (s *StartData) MarshalMap(m map[string]interface{}) error {
	m["cc_partition"] = s.cc_partition

	m["instance_index"] = s.instance_index

	m["application_version"] = s.application_version
	m["application_name"] = s.application_name
	m["application_uris"] = s.application_uris

	m["prod"] = s.prod

	m["application_id"] = s.application_id
	m["droplet_sha1"] = s.droplet_sha1

	m["droplet_uri"] = s.droplet_uri

	m["start_command"] = s.start_command

	m["environment"] = s.environment

	limit := make(map[string]uint64)
	m["limits"] = limit
	s.limits.MarshalMap(limit)

	services := make([]map[string]interface{}, len(s.services))
	m["services"] = services
	for i, s := range s.services {
		s.MarshalMap(services[i])
	}

	return nil
}

func (l *LimitsData) UnmarshalMap(m map[string]interface{}) error {
	l.memMb = getUInt(m, "mem")
	l.diskMb = getUInt(m, "disk")
	l.fds = getUInt(m, "fds")

	return nil
}

func (l *LimitsData) MarshalMap(m map[string]uint64) error {
	m["mem"] = uint64(l.memMb)
	m["disk"] = uint64(l.diskMb)
	m["fds"] = l.fds

	return nil
}

func (s *ServiceData) UnmarshalMap(m map[string]interface{}) error {
	s.name = getString(m, "name")
	s.label = getString(m, "label")
	s.credentials = m["credentials"].(map[string]interface{})
	s.tags = m["tags"].([]string)
	s.syslog_drain_url = getString(m, "syslog_drain_url")

	if opts, exists := m["options"]; exists {
		s.options = opts.(map[string]interface{})
	}

	s.provider = getString(m, "provider")
	s.vendor = getString(m, "vendor")
	s.plan = getString(m, "plan")

	s.version = getString(m, "version")

	return nil
}

func (s *ServiceData) MarshalMap(m map[string]interface{}) error {
	m["name"] = s.name
	m["label"] = s.label
	m["credentials"] = s.credentials
	m["tags"] = s.tags
	m["syslog_drain_url"] = s.syslog_drain_url

	m["options"] = s.options

	m["provider"] = s.provider
	m["vendor"] = s.vendor
	m["plan"] = s.plan

	m["version"] = s.version

	return nil
}

func getBool(m map[string]interface{}, key string) bool {
	if b, exist := m[key]; exist {
		return b.(bool)
	}
	return false
}

func getString(m map[string]interface{}, key string) string {
	if b, exist := m[key]; exist {
		return b.(string)
	}
	return ""
}

func getUInt(m map[string]interface{}, key string) uint64 {
	if i, exist := m[key]; exist {
		rv := reflect.ValueOf(i)
		switch i.(type) {
		case int, int8, int32, int64:
			return uint64(rv.Int())
		case uint, uint8, uint32, uint64:
			return rv.Uint()
		case float32, float64:
			return uint64(rv.Float())
		}
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	if i, exist := m[key]; exist {
		rv := reflect.ValueOf(i)
		switch i.(type) {
		case int, int8, int32, int64:
			return int(rv.Int())
		case uint, uint8, uint32, uint64:
			return int(rv.Uint())
		case float32, float64:
			return int(rv.Float())
		}
	}
	return 0
}
