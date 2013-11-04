package directory_server

import (
	"dea/config"
	"dea/staging"
	"dea/starting"
	"dea/utils"
	"directoryserver"
	"errors"
	"fmt"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/nu7hatch/gouuid"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var verifiable_file_params = []string{"path", "timestamp"}
var required_query_params = []string{"hmac", "path", "timestamp"}

type hmacDirectoryServer interface {
	listPath(w http.ResponseWriter, r *http.Request, path string)
	verify_instance_file_url(u *url.URL, params url.Values) bool
}

type DirectoryServerV2 struct {
	protocol      string
	uuid          *uuid.UUID
	domain        string
	localIp       string
	port          uint16
	dirConfig     config.DirServerConfig
	listener      net.Listener
	instancepaths *instancePaths
	stagingtasks  *stagingTasks
	hmacHelper    *HMACHelper
	logger        *steno.Logger
}

func NewDirectoryServerV2(localIp string, domain string, config config.DirServerConfig) (*DirectoryServerV2, error) {
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
		port:       config.V2Port,
		hmacHelper: NewHMACHelper(hmacKey[:]),
		logger:     utils.Logger("directoryserver_v2", nil),
	}, nil
}

func (ds *DirectoryServerV2) Port() uint16 {
	return ds.port
}

func (ds *DirectoryServerV2) Configure_endpoints(instanceRegistry *starting.InstanceRegistry, stagingTaskRegistry *staging.StagingTaskRegistry) {
	ds.instancepaths = newInstancePaths(instanceRegistry, ds, 60*60)
	ds.stagingtasks = newStagingTasks(stagingTaskRegistry, ds, 60*60)
}

func (ds *DirectoryServerV2) Start() error {
	address := ds.localIp + ":" + strconv.Itoa(int(ds.port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	ds.listener = listener

	msg := fmt.Sprintf("Starting HTTP server at host:port %s", address)
	ds.logger.Info(msg)

	go http.Serve(listener, ds)
	return nil
}

func (ds *DirectoryServerV2) Stop() error {
	return ds.listener.Close()
}

// If validation with the DEA is successful, the HTTP request is served.
// Otherwise, the same HTTP response from the DEA is served as response to
// the HTTP request.
func (ds *DirectoryServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasPrefix(path, instance_paths) {
		ds.instancepaths.ServeHTTP(w, r)
	} else if strings.HasPrefix(path, staging_tasks) {
		ds.stagingtasks.ServeHTTP(w, r)
	}
}

func queryParamsExists(keys []string, values url.Values) bool {
	for _, k := range keys {
		if _, ok := values[k]; !ok {
			return false
		}
	}
	return true
}

// Lists directory, or writes file contents in the HTTP response as per the
// the response received from the DEA. If the "tail" parameter is part of
// the HTTP request, then the file contents are streamed through chunked
// HTTP transfer encoding. Otherwise, the entire file is dumped in the HTTP
// response.
//
// Writes appropriate errors and status codes in the HTTP response if there is
// a problem in reading the file or directory.
func (ds *DirectoryServerV2) listPath(w http.ResponseWriter, r *http.Request, path string) {
	info, err := os.Stat(path)
	if err != nil {
		ds.logger.Warnf("%s", err)
		entity_not_found(w)
		return
	}

	if info.IsDir() {
		list_directory(path, r.URL.String(), w)
	} else {
		ds.writeFile(r, w, path)
	}
}

func (ds *DirectoryServerV2) writeFile(request *http.Request, writer http.ResponseWriter, path string) {
	var err error

	if ds.shouldTail(request) {
		err = ds.tailFile(request, writer, path)
	} else {
		err = ds.dumpFile(request, writer, path)
	}

	if err != nil {
		writeServerError(&err, writer)
	}
}

func (ds *DirectoryServerV2) shouldTail(request *http.Request) bool {
	_, present := request.URL.Query()["tail"]
	return present
}

func (ds *DirectoryServerV2) hasTailOffset(request *http.Request) bool {
	_, present := request.URL.Query()["tail_offset"]
	return present
}

func (ds *DirectoryServerV2) getTailOffset(request *http.Request) (int64, error) {
	param, present := request.URL.Query()["tail_offset"]
	if !present {
		return 0, errors.New("Offset not part of query.")
	}

	offset, err := strconv.Atoi(param[0])
	if err != nil || offset < 0 {
		return 0, errors.New("Tail offset must be a positive integer.")
	}

	return int64(offset), nil
}

func (ds *DirectoryServerV2) tailFile(request *http.Request, writer http.ResponseWriter,
	path string) error {

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	var offset int64
	var seekType int

	if ds.hasTailOffset(request) {
		seekType = os.SEEK_SET
		offset, err = ds.getTailOffset(request)
		if err != nil {
			return err
		}
	} else {
		seekType = os.SEEK_END
		offset = 0
	}

	_, err = file.Seek(offset, seekType)
	if err != nil {
		return err
	}

	fileStreamer := &directoryserver.StreamHandler{
		File:          file,
		FlushInterval: 50 * time.Millisecond,
		IdleTimeout:   time.Duration(ds.dirConfig.StreamingTimeout) * time.Second,
	}

	fileStreamer.ServeHTTP(writer, request)
	return nil
}

// Dumps the contents of the specified file in the HTTP response.
// Also handles HTTP byte range requests.
// Returns an error if there is a problem in opening/closing the file.
func (ds *DirectoryServerV2) dumpFile(request *http.Request, writer http.ResponseWriter,
	path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	handle, err := os.Open(path)
	if err != nil {
		return err
	}

	// This takes care of serving HTTP byte range request if present.
	// Otherwise dumps the entire file in the HTTP response.
	http.ServeContent(writer, request, path, info.ModTime(), handle)
	return handle.Close()
}

func (ds *DirectoryServerV2) Instance_file_url_for(instance_id, file_path string) string {
	path := fmt.Sprintf("/instance_paths/%s", instance_id)
	return ds.hmaced_url_for(path,
		map[string]string{
			"path":      file_path,
			"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		}, verifiable_file_params)
}

func (ds *DirectoryServerV2) staging_task_file_url_for(task_id, file_path string) string {
	path := fmt.Sprintf("/staging_tasks/%s/file_path", task_id)
	return ds.hmaced_url_for(path,
		map[string]string{
			"path":      file_path,
			"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		}, verifiable_file_params)
}

func (ds *DirectoryServerV2) External_hostname() string {
	return fmt.Sprintf("%s.%s", ds.uuid, ds.domain)
}

func (ds *DirectoryServerV2) verify_hmaced_url(u *url.URL, params url.Values, params_to_verify []string) bool {
	v := url.Values{}
	for _, k := range params_to_verify {
		if val, exists := params[k]; exists {
			v[k] = val
		}
	}

	hmacUrl := url.URL{Path: u.Path,
		RawQuery: v.Encode(),
	}

	return ds.hmacHelper.Compare([]byte(params.Get("hmac")), hmacUrl.String())
}

func (ds *DirectoryServerV2) verify_instance_file_url(u *url.URL, params url.Values) bool {
	return ds.verify_hmaced_url(u, params, verifiable_file_params)
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
