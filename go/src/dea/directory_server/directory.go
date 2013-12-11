package directory_server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func entity_not_found(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func forbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
}

func bad_request(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

func service_unavailable(w http.ResponseWriter) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

func list_directory(path, path_info string, w http.ResponseWriter) {

	root := len(strings.TrimLeft(path_info, "/")) == 0

	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		w.Header().Set("Content-Length", strconv.Itoa(len(err.Error())))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
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

		_, err := w.Write([]byte(fmt.Sprintf("%-35s %10s\n", basename, size)))
		if err != nil {
			return
		}
	}
}

// Writes the new error message in the HTTP response and sets the HTTP response
// status code to 500.
func writeServerErrorMessage(errorMessage string, w http.ResponseWriter) {
	msg := fmt.Sprintf("Can't serve request due to error: %s", errorMessage)
	http.Error(w, msg, http.StatusInternalServerError)
}

func writeServerError(err *error, w http.ResponseWriter) {
	writeServerErrorMessage((*err).Error(), w)
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
