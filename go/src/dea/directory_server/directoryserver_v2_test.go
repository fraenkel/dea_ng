package directory_server

import (
	cfg "dea/config"
	"dea/staging"
	"dea/starting"
	"dea/testhelpers"
	"dea/utils"
	"encoding/hex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
)

var _ = Describe("DirectoryserverV2", func() {
	var config *cfg.Config
	var server *DirectoryServerV2
	var instanceRegistry *starting.InstanceRegistry
	var instance *starting.Instance
	var stagingTaskRegistry *staging.StagingTaskRegistry
	var stagingTask *staging.StagingTask

	BeforeEach(func() {
		_, file, _, _ := runtime.Caller(0)
		cfgPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
		config, _ = cfg.ConfigFromFile(cfgPath)

		config.DirectoryServer.V2Port = 3456

		instanceRegistry = starting.NewInstanceRegistry(config)

		attrs := make(map[string]interface{})
		attrs["application_id"] = "appId"
		instance = starting.NewInstance(attrs, config, nil, "127.0.0.1")

		instanceRegistry.Register(instance)

		stagingTaskRegistry = staging.NewStagingTaskRegistry()
		stagingTask = staging.NewStagingTask(config, staging.NewStagingMessage(helpers.Valid_staging_attributes()),
			[]staging.StagingBuildpack{}, nil, utils.Logger("staging_tasks_test_logger", nil))

	})

	JustBeforeEach(func() {
		server, _ = NewDirectoryServerV2("127.0.0.1", "example.org", config.DirectoryServer)
	})

	Describe("configure_endpoints", func() {
		JustBeforeEach(func() {
			server.Configure_endpoints(instanceRegistry, stagingTaskRegistry)
		})

		It("sets up file api server", func() {
			Expect(server.address).To(Equal("127.0.0.1:3456"))
		})

		It("configures instance paths resource endpoints", func() {
			Expect(server.instancepaths.instanceRegistry).To(Equal(instanceRegistry))
			Expect(server.instancepaths.max_age_secs).To(Equal(int64(3600)))
			Expect(server.instancepaths.hmacServer).To(Equal(server))
		})

		It("configures staging tasks resource endpoints", func() {
			Expect(server.stagingtasks.stagingRegistry).To(Equal(stagingTaskRegistry))
			Expect(server.stagingtasks.max_age_secs).To(Equal(int64(3600)))
			Expect(server.stagingtasks.hmacServer).To(Equal(server))
		})
	})

	Describe("start", func() {
		Context("when file api server was configured", func() {
			BeforeEach(func() {
				// Pick a random port since we cannot stop a Server reliably yet
				config.DirectoryServer.V2Port = 0
			})
			JustBeforeEach(func() {
				server.Configure_endpoints(instanceRegistry, stagingTaskRegistry)
			})

			It("can handle instance paths requests", func() {
				u := server.Instance_file_url_for("instance-id", "some-file-path")

				err := server.Start()
				Expect(err).To(BeNil())

				getUrl := localize_url(u, server.address)
				rsp, err := http.Get(getUrl)
				server.Stop()
				Expect(err).To(BeNil())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})

			It("can handle staging tasks requests", func() {
				u := server.staging_task_file_url_for("task-id", "some-file-path")

				err := server.Start()
				Expect(err).To(BeNil())

				getUrl := localize_url(u, server.address)
				rsp, err := http.Get(getUrl)
				server.Stop()
				Expect(err).To(BeNil())
				Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
			})
		})
	})

	Describe("url generation", func() {

		Describe("hmaced_url_for", func() {
			var hmacURL string
			inputs := &hmacInputs{}

			JustBeforeEach(func() {
				fakeHMAC := &fakeHMACHelper{createReturn: []byte("hmac-value")}
				server.hmacHelper = fakeHMAC
				server.protocol = "FAKEPROTOCOL"
				hmacURL = server.hmaced_url_for("/path", map[string]string{"param": "value"}, []string{"param"})

				inputs.server = server
				inputs.fakeHMAC = fakeHMAC
				inputs.hmacURL = hmacURL
				inputs.path = "/path"
				inputs.message = "/path?param=value"

			})

			it_generates_url(inputs)
			it_hmacs_url(inputs)

			It("includes given params", func() {
				u, err := url.Parse(hmacURL)
				Expect(err).To(BeNil())
				Expect(u.Query().Get("param")).To(Equal("value"))
			})

			It("takes protocol from config", func() {
				Expect(hmacURL).To(MatchRegexp("^FAKEPROTOCOL://"))
			})
		})

		Describe("instance_file_url_for", func() {
			var instancefileURL string
			iInputs := &hmacInputs{}
			var timestamp string

			JustBeforeEach(func() {

				fakeHMAC := &fakeHMACHelper{createReturn: []byte("hmac-value")}
				server.hmacHelper = fakeHMAC
				server.protocol = "FAKEPROTOCOL"

				instancefileURL = server.Instance_file_url_for("instance-id", "/path-to-file")
				u, _ := url.Parse(instancefileURL)
				timestamp = u.Query().Get("timestamp")

				iInputs.server = server
				iInputs.fakeHMAC = fakeHMAC
				iInputs.hmacURL = instancefileURL

				iInputs.path = "/instance_paths/instance-id"
				iInputs.message = "/instance_paths/instance-id?path=%2Fpath-to-file&timestamp=" + timestamp
			})

			it_generates_url(iInputs)
			it_hmacs_url(iInputs)

			It("includes file path", func() {
				u, _ := url.Parse(instancefileURL)
				Expect(u.Query().Get("path")).To(Equal("/path-to-file"))
			})
		})

		Describe("staging_task_url_for", func() {
			var stagingtaskURL string
			iInputs := &hmacInputs{}
			var timestamp string

			JustBeforeEach(func() {

				fakeHMAC := &fakeHMACHelper{createReturn: []byte("hmac-value")}
				server.hmacHelper = fakeHMAC
				server.protocol = "FAKEPROTOCOL"

				stagingtaskURL = server.staging_task_file_url_for("task-id", "/path-to-file")
				u, _ := url.Parse(stagingtaskURL)
				timestamp = u.Query().Get("timestamp")

				iInputs.server = server
				iInputs.fakeHMAC = fakeHMAC
				iInputs.hmacURL = stagingtaskURL

				iInputs.path = "/staging_tasks/task-id/file_path"
				iInputs.message = "/staging_tasks/task-id/file_path?path=%2Fpath-to-file&timestamp=" + timestamp
			})

			it_generates_url(iInputs)
			it_hmacs_url(iInputs)

			It("includes file path", func() {
				u, _ := url.Parse(stagingtaskURL)
				Expect(u.Query().Get("path")).To(Equal("/path-to-file"))
			})
		})
	})

	Describe("verify_hmaced_url", func() {
		Context("when hmac-ed path matches original path", func() {
			It("returns true", func() {
				verified_params := []string{}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param": "value"}, verified_params)
				u, _ := url.Parse(hmacUrl)
				Expect(server.verify_hmaced_url(u, u.Query(), verified_params)).To(BeTrue())
			})
		})

		Context("when path does not match original path", func() {
			It("returns false", func() {
				verified_params := []string{}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param": "value"}, verified_params)
				u, _ := url.Parse(hmacUrl)
				u.Path = "/malicious-path"
				Expect(server.verify_hmaced_url(u, u.Query(), verified_params)).To(BeFalse())
			})
		})

		Context("when hmac-ed params match original params", func() {
			It("returns true when verifying all params", func() {
				verified_params := []string{"param1", "param2"}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param1": "value1", "param2": "value2"},
					verified_params)
				u, _ := url.Parse(hmacUrl)
				Expect(server.verify_hmaced_url(u, u.Query(), verified_params)).To(BeTrue())
			})

			It("returns true when verifying specific params", func() {
				verified_params := []string{"param1"}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param1": "value1", "param2": "value2"},
					verified_params)
				u, _ := url.Parse(hmacUrl)
				Expect(server.verify_hmaced_url(u, u.Query(), verified_params)).To(BeTrue())
			})

		})

		Context("when hmac-ed param does not match original param", func() {
			It("returns false", func() {
				verified_params := []string{"param"}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param": "value"}, verified_params)
				u, _ := url.Parse(hmacUrl)
				query := u.Query()
				query.Set("param", "malicious-value")
				Expect(server.verify_hmaced_url(u, query, verified_params)).To(BeFalse())
			})
		})

		Context("when non-hmac-ed param is added (to support misc params additions)", func() {
			It("returns true", func() {
				verified_params := []string{"param"}
				hmacUrl := server.hmaced_url_for("/path", map[string]string{"param": "value"}, verified_params)
				u, _ := url.Parse(hmacUrl)
				query := u.Query()
				query.Set("new_param", "new-value")
				Expect(server.verify_hmaced_url(u, query, verified_params)).To(BeTrue())
			})
		})

		Context("when url does not have hmac param", func() {
			It("returns false", func() {
				verified_params := []string{}
				u, _ := url.Parse("http://google.com")
				Expect(server.verify_hmaced_url(u, u.Query(), verified_params)).To(BeFalse())
			})
		})

	})

})

type fakeHMACHelper struct {
	createMessage string
	createReturn  []byte
}

func (hmac *fakeHMACHelper) Create(message string) []byte {
	hmac.createMessage = message
	return hmac.createReturn
}

func (hmac *fakeHMACHelper) Compare(correct_hmac []byte, message string) bool {
	return true
}

func localize_url(u string, newHost string) string {
	newU, _ := url.Parse(u)
	newU.Host = newHost
	return newU.String()
}

type hmacInputs struct {
	server   *DirectoryServerV2
	fakeHMAC *fakeHMACHelper
	hmacURL  string
	path     string
	message  string
}

func it_generates_url(inputs *hmacInputs) {
	It("includes external host", func() {
		Expect(inputs.hmacURL).To(MatchRegexp("^" + inputs.server.protocol + "://" +
			inputs.server.uuid.String() + "." + inputs.server.domain))
	})

	It("includes path", func() {
		Expect(inputs.hmacURL).To(ContainSubstring(inputs.server.domain + inputs.path))
	})
}

func it_hmacs_url(inputs *hmacInputs) {
	It("includes generated hmac param", func() {
		Expect(inputs.message).To(Equal(inputs.fakeHMAC.createMessage))
		Expect(inputs.hmacURL).To(ContainSubstring(hex.EncodeToString([]byte("hmac-value"))))
	})
}
