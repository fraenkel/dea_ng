package starting

import (
	"errors"
	"reflect"
	"strings"
)

type LimitsData struct {
	MemMb  uint64
	DiskMb uint64
	Fds    uint64
}

type ServiceData struct {
	Name             string
	Label            string
	Credentials      map[string]interface{}
	Tags             []string
	Syslog_drain_url string

	Options map[string]interface{}

	Provider string
	Vendor   string
	Plan     string

	Version string
}

type StartData struct {
	CC_partition string

	Instance_index int

	Application_id      string
	Application_version string
	Application_name    string
	Application_uris    []string

	Prod bool

	Droplet_sha1 string
	Droplet_uri  string

	Start_command string

	LimitsData LimitsData

	Env          map[string]string
	ServicesData []ServiceData
}

func NewStartData(m map[string]interface{}) (StartData, error) {
	sd := StartData{}
	if err := sd.UnmarshalMap(m); err != nil {
		return sd, err
	}

	return sd, nil
}

func (s StartData) Index() int {
	return s.Instance_index
}

func (s StartData) Limits() map[string]uint64 {
	limit := make(map[string]uint64)
	s.LimitsData.MarshalMap(limit)
	return limit
}

func (s StartData) MemoryLimit() uint64 {
	return uint64(s.LimitsData.MemMb)
}

func (s StartData) Name() string {
	return s.Application_name
}

func (s StartData) Version() string {
	return s.Application_version
}
func (s StartData) Uris() []string {
	return s.Application_uris
}

func (s StartData) Services() []map[string]interface{} {
	services := make([]map[string]interface{}, len(s.ServicesData))
	for i, s := range s.ServicesData {
		service := make(map[string]interface{})
		s.MarshalMap(service)
		services[i] = service
	}
	return services
}

func (s StartData) Environment() map[string]string {
	return s.Env
}

func (s *StartData) UnmarshalMap(m map[string]interface{}) error {
	s.CC_partition = m["cc_partition"].(string)

	s.Instance_index = getInt(m, "instance_index")

	s.Application_version = getString(m, "application_version")
	if s.Application_version == "" {
		s.Application_version = getString(m, "version")
	}
	s.Application_name = getString(m, "application_name")
	if s.Application_name == "" {
		s.Application_name = getString(m, "name")
	}

	uris := getStrings(m, "application_uris")
	if uris == nil {
		uris = getStrings(m, "uris")
	}
	if uris == nil {
		uris = []string{}
	}

	s.Application_uris = uris

	s.Prod = getBool(m, "prod")

	s.Application_id = getString(m, "application_id")
	if s.Application_id == "" {
		s.Application_id = getString(m, "droplet")
	}

	s.Droplet_sha1 = getString(m, "droplet_sha1")
	if s.Droplet_sha1 == "" {
		s.Droplet_sha1 = getString(m, "sha1")
	}

	s.Droplet_uri = getString(m, "droplet_uri")
	if s.Droplet_uri == "" {
		s.Droplet_uri = getString(m, "executableUri")
	}

	s.Start_command = getString(m, "start_command")

	if env, exist := m["environment"]; exist {
		s.Env = env.(map[string]string)
	} else {
		env := make(map[string]string)
		s.Env = env
		rv := reflect.ValueOf(m["env"])
		if !rv.IsNil() && (rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice) {
			for i := 0; i < rv.Len(); i++ {
				elemV := rv.Index(i)
				if elemV.Kind() == reflect.Interface {
					elemV = elemV.Elem()
				}
				pair := strings.SplitN(elemV.String(), "=", 2)
				val := ""
				if len(pair) == 2 {
					val = pair[1]
				}
				env[pair[0]] = val
			}
		}
	}

	if err := s.LimitsData.UnmarshalMap(m["limits"]); err != nil {
		return err
	}

	var serviceData []map[string]interface{}
	switch m["services"].(type) {
	case []interface{}:
		ss := m["services"].([]interface{})
		serviceData = make([]map[string]interface{}, len(ss))
		for _, v := range ss {
			sd := v.(map[string]interface{})
			serviceData = append(serviceData, sd)
		}
	case []map[string]interface{}:
		serviceData = m["services"].([]map[string]interface{})
	default:
		return errors.New("Unexpected service data type: " + reflect.ValueOf(m["services"]).Kind().String())
	}

	s.ServicesData = make([]ServiceData, len(serviceData))
	for i := 0; i < len(serviceData); i++ {
		if err := s.ServicesData[i].UnmarshalMap(serviceData[i]); err != nil {
			return err
		}
	}

	return nil
}

func (s *StartData) MarshalMap(m map[string]interface{}) error {
	m["cc_partition"] = s.CC_partition

	m["instance_index"] = s.Instance_index

	m["application_version"] = s.Application_version
	m["application_name"] = s.Application_name
	m["application_uris"] = s.Application_uris

	m["prod"] = s.Prod

	m["application_id"] = s.Application_id
	m["droplet_sha1"] = s.Droplet_sha1

	m["droplet_uri"] = s.Droplet_uri

	m["start_command"] = s.Start_command

	m["environment"] = s.Env

	limit := make(map[string]uint64)
	m["limits"] = limit
	s.LimitsData.MarshalMap(limit)

	services := make([]map[string]interface{}, len(s.ServicesData))
	m["services"] = services
	for i, s := range s.ServicesData {
		smap := make(map[string]interface{})
		s.MarshalMap(smap)
		services[i] = smap
	}

	return nil
}

func (l *LimitsData) UnmarshalMap(m interface{}) error {
	switch m.(type) {
	case map[string]uint64:
		umap := m.(map[string]uint64)
		l.MemMb = umap["mem"]
		l.DiskMb = umap["disk"]
		l.Fds = umap["fds"]
	case map[string]interface{}:
		imap := m.(map[string]interface{})
		l.MemMb = getUInt(imap, "mem")
		l.DiskMb = getUInt(imap, "disk")
		l.Fds = getUInt(imap, "fds")
	default:
		return &reflect.ValueError{"MarshalMap", reflect.ValueOf(m).Kind()}
	}

	return nil
}

func (l *LimitsData) MarshalMap(m interface{}) error {
	switch m.(type) {
	case map[string]uint64:
		umap := m.(map[string]uint64)
		umap["mem"] = uint64(l.MemMb)
		umap["disk"] = uint64(l.DiskMb)
		umap["fds"] = l.Fds

	case map[string]interface{}:
		imap := m.(map[string]interface{})
		imap["mem"] = uint64(l.MemMb)
		imap["disk"] = uint64(l.DiskMb)
		imap["fds"] = l.Fds

	default:
		return &reflect.ValueError{"MarshalMap", reflect.ValueOf(m).Kind()}
	}

	return nil
}

func (s *ServiceData) UnmarshalMap(m map[string]interface{}) error {
	s.Name = getString(m, "name")
	s.Label = getString(m, "label")
	if creds, exists := m["credentials"]; exists {
		s.Credentials = creds.(map[string]interface{})
	}

	s.Tags = getStrings(m, "tags")

	s.Syslog_drain_url = getString(m, "syslog_drain_url")

	if opts, exists := m["options"]; exists {
		s.Options = opts.(map[string]interface{})
	}

	s.Provider = getString(m, "provider")
	s.Vendor = getString(m, "vendor")
	s.Plan = getString(m, "plan")

	s.Version = getString(m, "version")

	return nil
}

func (s *ServiceData) MarshalMap(m map[string]interface{}) error {
	m["name"] = s.Name
	m["label"] = s.Label
	m["credentials"] = s.Credentials
	m["tags"] = s.Tags
	m["syslog_drain_url"] = s.Syslog_drain_url

	m["options"] = s.Options

	m["provider"] = s.Provider
	m["vendor"] = s.Vendor
	m["plan"] = s.Plan

	m["version"] = s.Version

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
		switch b.(type) {
		case string:
			return b.(string)
		}
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

func getStrings(m map[string]interface{}, key string) []string {
	var ret []string
	if s, exist := m[key]; exist {
		switch s.(type) {
		case []string:
			ret = s.([]string)
		case []interface{}:
			is := s.([]interface{})
			ret = make([]string, len(is))
			for i := 0; i < len(is); i++ {
				ret[i] = is[i].(string)
			}
		}
	}
	return ret
}
