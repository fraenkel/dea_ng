package directory_server

import (
	cfg "dea/config"
	"dea/container"
	"dea/starting"
	"dea/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var _ = Describe("InstancePaths", func() {
	var config *cfg.Config
	var server *DirectoryServerV2
	var instanceRegistry starting.InstanceRegistry
	var instance *starting.Instance

	BeforeEach(func() {
		_, file, _, _ := runtime.Caller(0)
		cfgPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
		c, _ := cfg.LoadConfig(cfgPath)
		config = &c

		instanceRegistry = starting.NewInstanceRegistry(config)

		attrs := testhelpers.Valid_instance_attributes(false)
		instance = starting.NewInstance(attrs, config, nil, "127.0.0.1")

		instanceRegistry.Register(instance)
	})

	JustBeforeEach(func() {
		server, _ = NewDirectoryServerV2("127.0.0.1", "example.org", config.DirectoryServer)
		server.Configure_endpoints(instanceRegistry, nil)
		server.instancepaths.max_age_secs = 1
	})

	Describe("GET /instance_paths/<instance_id>", func() {
		It("returns a 401 if the hmac is missing", func() {
			w := instancePathRequest(server, instance.Id(), "/path",
				map[string]interface{}{"hmac": ""})

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns a 400 if the timestamp is too old", func() {
			u := instance_path(server, instance.Id(), "/path", nil)

			time.Sleep(2 * time.Second)
			req, err := http.NewRequest("GET", u, nil)
			w := httptest.NewRecorder()
			server.instancepaths.ServeHTTP(w, req)
			Expect(err).To(BeNil())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns a 404 if no instance exists", func() {
			w := instancePathRequest(server, "non-existant-id", "/path", nil)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		Context("when instance path is not available", func() {
			BeforeEach(func() {
				instance.Container = &container.MockContainer{MPath: ""}
			})

			It("returns a 503 if the instance path is unavailable", func() {
				w := instancePathRequest(server, instance.Id(), "/path", nil)
				Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
			})
		})
		Context("when instance path is available", func() {
			var tmpdir string

			BeforeEach(func() {
				tmpdir, _ = ioutil.TempDir("", "instance_paths")
				// resolve tmpdir properly on OSX
				tmpdir, _ = filepath.EvalSymlinks(tmpdir)
				home_dir := filepath.Join(tmpdir, "tmp", "rootfs", "home", "vcap")
				os.MkdirAll(home_dir, 0755)
				file, _ := os.Create(filepath.Join(home_dir, "test"))
				file.Close()

				instance.Container = &container.MockContainer{MPath: tmpdir}
				instance.SetState(starting.STATE_CRASHED)
			})

			AfterEach(func() {
				os.RemoveAll(tmpdir)
			})

			It("returns 404 if the requested file doesn't exist", func() {
				w := instancePathRequest(server, instance.Id(), "/unknown-path", nil)
				Expect(w.Code).To(Equal(http.StatusNotFound))
			})

			It("returns 403 if the file points outside the instance directory", func() {
				w := instancePathRequest(server, instance.Id(), "/..", nil)
				Expect(w.Code).To(Equal(http.StatusForbidden))
			})

			It("returns 200 with full path on success", func() {
				w := instancePathRequest(server, instance.Id(), "/test", nil)
				Expect(w.Code).To(Equal(http.StatusOK))
			})

			It("return 200 with instance path when path is not explicitly specified (nil)", func() {
				w := instancePathRequest(server, instance.Id(), "", nil)
				Expect(w.Code).To(Equal(http.StatusOK))
			})

		})
	})
})

func instancePathRequest(server *DirectoryServerV2, id string, path string,
	options map[string]interface{}) *httptest.ResponseRecorder {
	u := instance_path(server, id, path, options)

	req, _ := http.NewRequest("GET", u, nil)
	w := httptest.NewRecorder()
	server.instancepaths.ServeHTTP(w, req)
	return w
}

func instance_path(server *DirectoryServerV2, instance_id string, file_path string, options map[string]interface{}) string {
	urlPath := server.Instance_file_url_for(instance_id, file_path)
	if options["hmac"] != nil {
		u, _ := url.Parse(urlPath)
		params := u.Query()
		params.Set("hmac", options["hmac"].(string))
		u.RawQuery = params.Encode()
		urlPath = u.String()
	}
	return urlPath
}
