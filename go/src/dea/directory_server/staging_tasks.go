package directory_server

import (
	"dea/staging"
	"dea/utils"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var staging_tasks = "/staging_tasks/"

type stagingTasks struct {
	stagingRegistry *staging.StagingTaskRegistry
	hmacServer      hmacDirectoryServer
	max_age_secs    int64
}

//GET /staging_tasks/<task_id>/file_path
func newStagingTasks(registry *staging.StagingTaskRegistry, hmacServer hmacDirectoryServer, max_url_age_secs int64) *stagingTasks {
	return &stagingTasks{
		stagingRegistry: registry,
		hmacServer:      hmacServer,
		max_age_secs:    max_url_age_secs,
	}
}

func (st *stagingTasks) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	if r.Method != "GET" || !queryParamsExists(required_query_params, params) {
		bad_request(w)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 && parts[3] != "file_path" {
		bad_request(w)
		return
	}

	st.get(parts[2], params, w, r)
}

func (st *stagingTasks) isExpired(timestamp int64) bool {
	url_age_secs := time.Now().Unix() - timestamp
	return url_age_secs > st.max_age_secs
}

func (st *stagingTasks) get(taskId string, params url.Values, w http.ResponseWriter, r *http.Request) {
	if !st.hmacServer.verify_instance_file_url(r.URL, params) {
		unauthorized(w)
		return
	}

	timestamp, err := strconv.ParseInt(params.Get("timestamp"), 10, 64)
	if err != nil {
		bad_request(w)
		return
	}

	if st.isExpired(timestamp) {
		bad_request(w)
		return
	}

	sTask := st.stagingRegistry.Task(taskId)
	if sTask == nil {
		entity_not_found(w)
		return
	}

	full_path := sTask.Path_in_container(params.Get("path"))
	if full_path == "" {
		service_unavailable(w)
		return
	}

	if !utils.File_Exists(full_path) {
		entity_not_found(w)
		return
	}

	real_path, err := filepath.EvalSymlinks(full_path)

	if err != nil || !strings.HasPrefix(real_path, full_path) {
		forbidden(w)
		return
	}

	st.hmacServer.listPath(w, r, real_path)
}
