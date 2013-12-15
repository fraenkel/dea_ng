package directory_server

import (
	"dea"
	cfg "dea/config"
	"dea/staging"
	"dea/testhelpers"
	tstaging "dea/testhelpers/staging"
	steno "github.com/cloudfoundry/gosteno"
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

var _ = Describe("StagingTasks", func() {
	var config *cfg.Config
	var server *DirectoryServerV2
	var stagingTaskRegistry *staging.StagingTaskRegistry
	var stagingTask *tstaging.FakeStagingTask

	BeforeEach(func() {
		_, file, _, _ := runtime.Caller(0)
		cfgPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
		c, _ := cfg.LoadConfig(cfgPath)
		config = &c

		stagingTaskRegistry = staging.NewStagingTaskRegistry(config, nil, staging.NewStagingTask)
		stagingTask = &tstaging.FakeStagingTask{StagingMsg: staging.NewStagingMessage(testhelpers.Valid_staging_attributes())}
	})

	JustBeforeEach(func() {
		server, _ = NewDirectoryServerV2("127.0.0.1", "example.org", nil, config.DirectoryServer)
		server.Configure_endpoints(nil, stagingTaskRegistry)
		server.stagingtasks.max_age_secs = 1
	})

	Describe("GET /staging_tasks/<task_id>/file_path", func() {
		Context("when hmac is missing", func() {
			It("returns a 401", func() {
				w := stagingtaskRequest(server, stagingTask.Id(), "file-path",
					map[string]interface{}{"hmac": ""})
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
		Context("when hmac is invalid", func() {
			It("returns a 401", func() {
				w := stagingtaskRequest(server, stagingTask.Id(), "file-path",
					map[string]interface{}{"hmac": "some-other-hmac"})
				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when url has expired", func() {
			It("returns a 400", func() {
				u := staging_task_file_path(server, stagingTask.Id(), "file-path", nil)

				time.Sleep(2 * time.Second)
				req, err := http.NewRequest("GET", u, nil)
				w := httptest.NewRecorder()
				server.stagingtasks.ServeHTTP(w, req)
				Expect(err).To(BeNil())
				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when staging task does not exist", func() {
			It("returns a 404", func() {
				w := stagingtaskRequest(server, "nonexistant-task-id", "file-path", nil)
				Expect(w.Code).To(Equal(http.StatusNotFound))
			})
		})

		Context("when staging task does exist", func() {
			BeforeEach(func() {
				stagingTaskRegistry.Register(stagingTask)
			})

			Context("when container path is not available", func() {
				BeforeEach(func() {
					stagingTask.ContainerPath = ""
				})

				It("returns a 503", func() {
					w := stagingtaskRequest(server, stagingTask.Id(), "file-path", nil)
					Expect(w.Code).To(Equal(http.StatusServiceUnavailable))
				})
			})

			Context("when container path does exist", func() {
				var tmpdir string

				BeforeEach(func() {
					tmpdir, _ = ioutil.TempDir("", "staging_tasks")
					// resolve tmpdir properly on OSX
					tmpdir, _ = filepath.EvalSymlinks(tmpdir)
					container_rootfs_path := filepath.Join(tmpdir, "tmp", "rootfs")
					os.MkdirAll(container_rootfs_path, 0755)
					stagingTask.ContainerPath = tmpdir
				})

				AfterEach(func() {
					os.RemoveAll(tmpdir)
				})

				Context("when requested file does not exist", func() {
					It("returns 404", func() {
						w := stagingtaskRequest(server, stagingTask.Id(), "file-path-that-does-not-exist", nil)
						Expect(w.Code).To(Equal(http.StatusNotFound))
					})
				})

				Context("when requested file path points outside the container's directory", func() {
					It("returns 403", func() {
						w := stagingtaskRequest(server, stagingTask.Id(), "..", nil)
						Expect(w.Code).To(Equal(http.StatusForbidden))
					})
				})

				Context("when file exists", func() {
					BeforeEach(func() {
						file, _ := os.Create(filepath.Join(tmpdir, "tmp", "rootfs", "some-file"))
						file.Close()
					})

					It("returns 200", func() {
						w := stagingtaskRequest(server, stagingTask.Id(), "some-file", nil)
						Expect(w.Code).To(Equal(http.StatusOK))
					})
				})
			})
		})
	})
})

func stagingtaskRequest(server *DirectoryServerV2, task_id string, file_path string, options map[string]interface{}) *httptest.ResponseRecorder {
	u := staging_task_file_path(server, task_id, file_path, options)

	req, _ := http.NewRequest("GET", u, nil)
	w := httptest.NewRecorder()
	server.stagingtasks.ServeHTTP(w, req)
	return w
}

func staging_task_file_path(server *DirectoryServerV2, task_id string, file_path string, options map[string]interface{}) string {
	urlPath := server.UrlForStagingTask(task_id, file_path)
	if options["hmac"] != nil {
		u, _ := url.Parse(urlPath)
		params := u.Query()
		params.Set("hmac", options["hmac"].(string))
		u.RawQuery = params.Encode()
		urlPath = u.String()
	}
	return urlPath
}

func NewMockStagingTask(*cfg.Config, staging.StagingMessage, []dea.StagingBuildpack, dea.DropletRegistry, *steno.Logger) dea.StagingTask {
	return &tstaging.FakeStagingTask{}
}
