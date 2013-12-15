package staging

import (
	"dea"
	thelpers "dea/testhelpers"
	"dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
)

var _ = Describe("AdminBuildpackDownloader", func() {
	var downloader AdminBuildpackDownloader
	var destination string
	var buildpacks []dea.StagingBuildpack
	var httpServer *httptest.Server

	BeforeEach(func() {
		destination, _ = ioutil.TempDir("", "adminbpdownload")
		httpServer = thelpers.NewFileServer()
	})

	JustBeforeEach(func() {
		downloader = NewAdminBuildpackDownloader(buildpacks, destination)
	})

	AfterEach(func() {
		os.RemoveAll(destination)
		if httpServer != nil {
			httpServer.Close()
		}
	})

	Context("with single buildpack", func() {
		BeforeEach(func() {
			url1, _ := url.Parse(httpServer.URL + "/buildpacks/uri/abcdef")
			buildpacks = []dea.StagingBuildpack{{Url: url1, Key: "abcdef"}}
		})

		It("downloads the buildpack and unzip it", func() {
			downloader.Download()

			expected_file_name := path.Join(destination, "abcdef")
			fileinfo, err := os.Stat(expected_file_name)
			Expect(os.IsNotExist(err)).To(BeFalse())
			Expect(fileinfo.Mode().Perm()).To(Equal(os.FileMode(0755)))

			expected_file_name = path.Join(destination, "abcdef", "content")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())
		})

		It("doesn't download buildpacks it already has", func() {
			download_dir := path.Join(destination, "abcdef")
			os.Mkdir(download_dir, 0755)
			downloader.Download()

			expected_file_name := path.Join(destination, "abcdef", "content")
			Expect(utils.File_Exists(expected_file_name)).To(BeFalse())
		})

	})

	Context("with multiple buildpack", func() {
		BeforeEach(func() {
			url1, _ := url.Parse(httpServer.URL + "/buildpacks/uri/abcdef")
			url2, _ := url.Parse(httpServer.URL + "/buildpacks/uri/ijgh")
			buildpacks = []dea.StagingBuildpack{
				{Url: url1, Key: "abcdef"},
				{Url: url2, Key: "ijgh"},
			}
		})

		It("only returns when all the downloads are done", func() {
			downloader.Download()

			expected_file_name := path.Join(destination, "abcdef", "content")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())

			expected_file_name = path.Join(destination, "ijgh", "content")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())

			children, _ := ioutil.ReadDir(destination)
			Expect(len(children)).Should(Equal(2))
		})

		Context("error handling", func() {
			BeforeEach(func() {
				buildpacks[1].Url.Path = thelpers.ERROR_PATH
			})

			It("doesn't throw exceptions if the download fails", func() {
				downloader.Download()

				expected_file_name := path.Join(destination, "abcdef", "content")
				Expect(utils.File_Exists(expected_file_name)).To(BeTrue())

				expected_file_name = path.Join(destination, "ijgh", "content")
				Expect(utils.File_Exists(expected_file_name)).To(BeFalse())

				children, _ := ioutil.ReadDir(destination)
				Expect(len(children)).Should(Equal(1))
			})
		})
	})

})
