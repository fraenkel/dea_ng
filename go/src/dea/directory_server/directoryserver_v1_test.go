package directory_server

import (
	cfg "dea/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"path/filepath"
	"runtime"
)

var _ = Describe("Directory", func() {
	var config cfg.Config
	var server *DirectoryServerV1

	BeforeEach(func() {
		_, file, _, _ := runtime.Caller(0)
		cfgPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../config/dea.yml"))
		config, _ = cfg.LoadConfig(cfgPath)
	})

	JustBeforeEach(func() {
		server, _ = NewDirectoryServerV1("127.0.0.1", 0,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
		server.Start()
	})

	AfterEach(func() {
		server.Stop()
	})

	It("should return a 401 with no credentials", func() {
		rsp, err := http.Get(server.Uri)
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("should return a 401 with partial credentials", func() {
		client := http.Client{}
		req, err := http.NewRequest("GET", server.Uri, nil)
		req.SetBasicAuth("Bob", "")
		rsp, err := client.Do(req)
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("should return a 404 for unknown instances", func() {
		client := http.Client{}
		req, err := http.NewRequest("GET", server.Uri+"garbage", nil)
		req.SetBasicAuth(server.Credentials[0], server.Credentials[1])
		rsp, err := client.Do(req)

		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("should return a 200 for a valid request", func() {
		client := http.Client{}
		req, err := http.NewRequest("GET", server.Uri+"/", nil)
		req.SetBasicAuth(server.Credentials[0], server.Credentials[1])
		rsp, err := client.Do(req)
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))
	})
})
