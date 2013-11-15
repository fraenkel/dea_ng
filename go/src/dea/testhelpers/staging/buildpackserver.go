package staging

import (
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
)

const ERROR_PATH = "/bad/path"

func NewFileServer() *httptest.Server {
	_, curfile, _, _ := runtime.Caller(0)
	buildpack := path.Join(curfile, "../../../../../fixtures/buildpack.zip")
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == ERROR_PATH {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		http.ServeFile(w, r, buildpack)
	}))

	return httpServer
}
