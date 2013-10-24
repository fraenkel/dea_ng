package directory_server

import (
	"dea/config"
	"dea/staging"
	"dea/starting"
	"dea/utils"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"net/url"
	"strconv"
	"time"
)

var verifiable_file_params = []string{"path", "timestamp"}

type DirectoryServerV2 struct {
	protocol   string
	uuid       *uuid.UUID
	domain     string
	port       uint16
	dirConfig  config.DirServerConfig
	hmacHelper *utils.HMACHelper
}

func NewDirectoryServerV2(domain string, port uint16, config config.DirServerConfig) (*DirectoryServerV2, error) {
	dirUuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	hmacKey, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &DirectoryServerV2{
		protocol:   config.Protocol,
		uuid:       dirUuid,
		domain:     domain,
		port:       port,
		hmacHelper: utils.NewHMACHelper(hmacKey[:]),
	}, nil
}

func (ds *DirectoryServerV2) Port() uint16 {
	return ds.port
}

func (ds *DirectoryServerV2) Configure_endpoints(instanceRegistry *starting.InstanceRegistry, stagingTaskRegistry *staging.StagingTaskRegistry) {
	panic("Not implemented")
}

func (ds *DirectoryServerV2) Start() {
	panic("Not implemented")
}

func (ds *DirectoryServerV2) Instance_file_url_for(instance_id, file_path string) string {
	path := fmt.Sprintf("/instance_paths/%s", instance_id)
	return ds.hmaced_url_for(path,
		map[string]string{
			"path":      file_path,
			"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		}, verifiable_file_params)
}

func (ds *DirectoryServerV2) External_hostname() string {
	return fmt.Sprintf("%s.%s", ds.uuid, ds.domain)
}

func (ds *DirectoryServerV2) hmaced_url_for(path string, params map[string]string,
	params_to_verify []string) string {
	v := url.Values{}
	for _, k := range params_to_verify {
		if val, exists := params[k]; exists {
			v.Set(k, val)
		}
	}

	verifiable_path_and_params := fmt.Sprintf("%s?%s", path, v.Encode())

	hmac := ds.hmacHelper.Create(verifiable_path_and_params)
	v.Set("hmac", string(hmac))
	params_with_hmac := v.Encode()

	return fmt.Sprintf("%s://%s%s?%s", ds.protocol, ds.External_hostname(), path, params_with_hmac)
}
