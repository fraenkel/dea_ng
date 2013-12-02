package directory_server

import (
	"dea/starting"
	"dea/utils"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var instance_paths = "/instance_paths/"

type instancePaths struct {
	instanceRegistry starting.InstanceRegistry
	hmacServer       hmacDirectoryServer
	max_age_secs     int64
}

//GET /instance_paths/<instance_id>
func newInstancePaths(registry starting.InstanceRegistry, hmacServer hmacDirectoryServer, max_url_age_secs int64) *instancePaths {
	return &instancePaths{
		instanceRegistry: registry,
		hmacServer:       hmacServer,
		max_age_secs:     max_url_age_secs,
	}
}

func (ip *instancePaths) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	if r.Method != "GET" || !queryParamsExists(required_query_params, params) {
		bad_request(w)
		return
	}

	instance_id := r.URL.Path[len(instance_paths):]
	ip.get(instance_id, params, w, r)
}

func (ip *instancePaths) isExpired(timestamp int64) bool {
	url_age_secs := time.Now().Unix() - timestamp
	return url_age_secs > ip.max_age_secs
}

func (ip *instancePaths) get(instance_id string, params url.Values, w http.ResponseWriter, r *http.Request) {

	if !ip.hmacServer.verify_instance_file_url(r.URL, params) {
		unauthorized(w)
		return
	}

	timestamp, err := strconv.ParseInt(params.Get("timestamp"), 10, 64)
	if err != nil {
		bad_request(w)
		return
	}

	if ip.isExpired(timestamp) {
		bad_request(w)
		return
	}

	instance := ip.instanceRegistry.LookupInstance(instance_id)
	if instance == nil {
		entity_not_found(w)
		return
	}

	if !instance.IsPathAvailable() {
		service_unavailable(w)
		return
	}

	iPath, err := instance.Path()
	if err != nil {
		entity_not_found(w)
		return
	}

	full_path := filepath.Join(iPath, params.Get("path"))
	if !utils.File_Exists(full_path) {
		entity_not_found(w)
		return
	}

	real_path, err := filepath.EvalSymlinks(full_path)
	if err != nil || !strings.HasPrefix(real_path, iPath) {
		forbidden(w)
		return
	}

	ip.hmacServer.listPath(w, r, real_path)
}
