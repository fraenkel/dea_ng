package directory_server

import (
	"dea/starting"
	"dea/utils"
	"fmt"
	steno "github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

type dirRequest struct {
	w         http.ResponseWriter
	r         *http.Request
	root      string
	path      string
	path_info string
}

type Directory struct {
	instanceRegistry *starting.InstanceRegistry
	logger           *steno.Logger
}

func NewDirectory(instanceRegistry *starting.InstanceRegistry) *Directory {
	return &Directory{
		instanceRegistry: instanceRegistry,
		logger:           utils.Logger("Directory", nil),
	}
}

func (dir *Directory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path_info := r.URL.String()
	path_parts := strings.Split(r.URL.Path, "/")

	dir.logger.Debug2f("Handling request %s", path_info)

	// Lookup container associated with request
	instance_id := path_parts[1]
	instance := dir.instanceRegistry.LookupInstance(instance_id)

	if instance == nil {
		dir.logger.Warnf("Unknown instance id: %s", instance_id)
		entity_not_found(w)
		return
	}

	if !instance.IsPathAvailable() {
		dir.logger.Warnf("Instance path unavailable for instance id: %s", instance_id)
		entity_not_found(w)
		return
	}

	dirReq := dirRequest{
		w: w,
		r: r,
	}

	// The instance path is the root for all future operations
	iPath, _ := instance.Path()
	dirReq.root, _ = filepath.EvalSymlinks(iPath)

	// Strip the instance id from the path. This is required to keep backwards
	// compatibility with how file URLs are constructed with DeaV1.
	dirReq.path_info = strings.Join(path_parts[2:], "/")
	dirReq.path = filepath.Clean(filepath.Join(dirReq.root, dirReq.path_info))

	if !utils.File_Exists(dirReq.path) {
		entity_not_found(w)
		return
	}

	resolve_sym_link(&dirReq)

	if isForbidden(dirReq) {
		dir.logger.Warnf("Path %s is forbidden", dirReq.path)
		forbidden(w)
		return
	}

	list_path(dirReq)
}

// TODO: add correct response if not readable, not sure if 404 is the best option
func list_path(dirReq dirRequest) {
	file, err := os.Open(dirReq.path)
	if err != nil {
		entity_not_found(dirReq.w)
		return
	}

	stat, err := file.Stat()
	file.Close()
	if err != nil {
		entity_not_found(dirReq.w)
		return
	}

	if stat.IsDir() {
		list_directory(dirReq)
	} else {
		http.ServeFile(dirReq.w, dirReq.r, dirReq.path)
	}

	return
}

func list_directory(dirReq dirRequest) {

	root := len(strings.TrimLeft(dirReq.path_info, "/")) == 0

	fileInfos, err := ioutil.ReadDir(dirReq.path)
	if err != nil {
		dirReq.w.WriteHeader(http.StatusInternalServerError)
		dirReq.w.Write([]byte(err.Error()))
		dirReq.w.Header().Set("Content-Length", strconv.Itoa(len(err.Error())))
		return
	}

	dirReq.w.WriteHeader(http.StatusOK)
	dirReq.w.Header().Set("Content-Type", "text/plain")
	for _, fileInfo := range fileInfos {
		basename := filepath.Base(fileInfo.Name())
		// ignore B29 control files, only return defaults
		if root && (basename != "app" && basename != "logs" && basename != "tomcat") {
			continue
		}

		var size string
		if fileInfo.IsDir() {
			size = "-"
			basename = basename + "/"
		} else {
			size = filesize_format(fileInfo.Size())
		}

		_, err := dirReq.w.Write([]byte(fmt.Sprintf("%-35s %10s\n", basename, size)))
		if err != nil {
			return
		}
	}
}

func isForbidden(dirReq dirRequest) bool {
	path := dirReq.path
	path_info := dirReq.path_info
	switch {
	case strings.Contains(path_info, ".."):
		return true
	case strings.HasSuffix(path_info, "/startup"):
		return true
	case strings.HasSuffix(path_info, "/stop"):
		return true
	}

	// breaks BVTs
	//forbidden = true if @path_info =~ /\/.+\/run\.pid/

	// Any symlink foolishness checked here
	check_path := strings.TrimRightFunc(path, unicode.IsSpace)
	path, _ = filepath.EvalSymlinks(path)

	if check_path != path {
		return true
	}

	return false
}

func resolve_sym_link(dirReq *dirRequest) {
	real_path, _ := filepath.EvalSymlinks(dirReq.path)
	if real_path == dirReq.path {
		return
	}

	// Adjust env only if user has access rights to real path
	app_base := filepath.Join(dirReq.root, strings.SplitN(strings.TrimLeft(dirReq.path_info, "/"), "/", 2)[0])
	if strings.HasPrefix(real_path, app_base) {
		idx := strings.Index(real_path, dirReq.root)
		if idx == -1 {
			return
		}

		// return the rest of the match
		dirReq.path_info = real_path[idx+len(dirReq.root):]
		dirReq.path = real_path
	}

	return
}

func entity_not_found(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func forbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

var filesizes = []string{"bytes", "KB", "MB", "GB", "TB"}

func filesize_format(filesize int64) string {
	num := float64(filesize)
	for _, size := range filesizes {
		if num < 1024.0 {
			return fmt.Sprintf("%.1f %s", num, size)
		}
		num = num / 1024.0
	}

	return ""
}
