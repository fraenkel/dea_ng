package directory_server

import (
	"bytes"
	cfg "dea/config"
	"dea/container"
	"dea/starting"
	"dea/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
)

var _ = Describe("Directory", func() {
	var config *cfg.Config
	var server *httptest.Server
	var instanceRegistry starting.InstanceRegistry
	var instance *starting.Instance
	var directory *Directory
	var tmpdir, app_dir string

	BeforeEach(func() {
		tmpdir, _ = ioutil.TempDir("", "directory_test")
		home_dir := path.Join(tmpdir, "tmp", "rootfs", "home", "vcap")
		os.MkdirAll(home_dir, 0755)

		app_dir = path.Join(home_dir, "app")
		os.MkdirAll(app_dir, 0755)

		logs_dir := path.Join(home_dir, "logs")
		os.MkdirAll(logs_dir, 0755)

		sentinel_path := path.Join(app_dir, "sentinel")
		sentinel_file, _ := os.Create(sentinel_path)
		sentinel_file.Write(bytes.Repeat([]byte("A"), 10000))
		sentinel_file.Close()

		config = &cfg.Config{}
		instanceRegistry = starting.NewInstanceRegistry(config)
		directory = NewDirectory(instanceRegistry)

		attrs := testhelpers.Valid_instance_attributes(false)

		instance = starting.NewInstance(attrs, config, nil, "127.0.0.1")

		instance.SetState(starting.STATE_CRASHED)
		instance.Container = &container.MockContainer{MPath: tmpdir}

		instanceRegistry.Register(instance)
	})

	JustBeforeEach(func() {
		server = httptest.NewServer(directory)
	})

	AfterEach(func() {
		server.Close()
		os.RemoveAll(tmpdir)
	})

	It("should return a 404 for unknown instances", func() {
		rsp, err := http.Get(server.URL + "/unknown")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("should return a 404 if the instance is unavailable", func() {
		instance.SetState(starting.STATE_STOPPED)
		Expect(instance.IsPathAvailable()).To(BeFalse())

		rsp, err := http.Get(server.URL + "/" + instance.Id())
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("should return a 404 if the requested file doesn't exist", func() {
		Expect(instance.IsPathAvailable()).To(BeTrue())

		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/nonexistant/path")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("should return file contents for files that exist", func() {
		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/app/sentinel")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))

		buf := make([]byte, 1000)
		total := 0
		for {
			n, err := rsp.Body.Read(buf)
			total = total + n
			if err != nil {
				break
			}
		}
		Expect(total).To(Equal(10000))
	})

	It("should list directories", func() {
		rsp, err := http.Get(server.URL + "/" + instance.Id())
		buf := make([]byte, 1000)
		rsp.Body.Read(buf)

		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))
		Expect(bytes.Contains(buf, []byte("app/"))).To(BeTrue())
		Expect(bytes.Contains(buf, []byte("logs/"))).To(BeTrue())

		rsp, err = http.Get(server.URL + "/" + instance.Id() + "/app")
		Expect(err).To(BeNil())

		n, _ := rsp.Body.Read(buf)
		buf = buf[:n]

		Expect(string(buf)).To(MatchRegexp(`sentinel\s+\d+\.?\d? K`))
	})

	It("should forbid requests containing relative path operators", func() {
		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/../..")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("should forbid requests for startup scripts", func() {
		file, err := os.Create(path.Join(app_dir, "startup"))
		file.Close()

		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/app/startup")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("should forbid requests for stop scripts", func() {
		file, err := os.Create(path.Join(app_dir, "stop"))
		file.Close()

		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/app/stop")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("should forbid requests for symlinks outside the container", func() {
		os.Symlink(tmpdir, path.Join(app_dir, "invalid_symlink"))

		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/app/invalid_symlink")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusForbidden))
	})

	It("should resolve symlinks inside the container", func() {
		src_path := path.Join(app_dir, "symlink_target")
		file, err := os.Create(src_path)
		file.Close()

		os.Symlink(src_path, path.Join(app_dir, "valid_symlink"))

		rsp, err := http.Get(server.URL + "/" + instance.Id() + "/app/valid_symlink")
		Expect(err).To(BeNil())
		Expect(rsp.StatusCode).To(Equal(http.StatusOK))
	})
})
