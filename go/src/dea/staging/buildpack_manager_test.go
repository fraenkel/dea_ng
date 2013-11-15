package staging_test

import (
	. "dea/staging"
	tstaging "dea/testhelpers/staging"
	"dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sort"
)

var _ = Describe("BuildpackManager", func() {
	var basedir string
	var admin_buildpacks_dir string
	var system_buildpacks_dir string
	var admin_buildpacks []StagingBuildpack
	var buildpacks_in_use []StagingBuildpack
	var manager BuildpackManager

	BeforeEach(func() {
		basedir, _ = ioutil.TempDir("", "buildpackmgr")
		os.MkdirAll(basedir, 0755)
		admin_buildpacks_dir = path.Join(basedir, "admin_buildpacks")
		system_buildpacks_dir = path.Join(basedir, "system_buildpacks")
	})

	AfterEach(func() {
		os.RemoveAll(basedir)
	})

	JustBeforeEach(func() {
		manager = NewBuildpackManager(admin_buildpacks_dir, system_buildpacks_dir, admin_buildpacks, buildpacks_in_use)
	})

	Describe("Download", func() {
		var httpServer *httptest.Server

		BeforeEach(func() {
			httpServer = tstaging.NewFileServer()

			url1, _ := url.Parse(httpServer.URL + "/buildpacks/uri/abcdef")
			admin_buildpacks = []StagingBuildpack{StagingBuildpack{Url: url1, Key: "abcdef"}}
			buildpacks_in_use = []StagingBuildpack{}
		})

		AfterEach(func() {
			httpServer.Close()
		})

		It("calls AdminBuildpackDownloader", func() {
			manager.Download()
			expected_file_name := path.Join(admin_buildpacks_dir, "abcdef")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())
		})
	})

	Describe("clean", func() {
		var file_to_delete string
		var file_to_keep string

		BeforeEach(func() {
			file_to_delete = path.Join(admin_buildpacks_dir, "1234")
			file_to_keep = path.Join(admin_buildpacks_dir, "abcdef")
			create_populated_directory(file_to_delete)
			create_populated_directory(file_to_keep)

			url1, _ := url.Parse("http://example.com/buildpacks/uri/abcdef")
			admin_buildpacks = []StagingBuildpack{StagingBuildpack{Url: url1, Key: "abcdef"}}
			buildpacks_in_use = []StagingBuildpack{}

		})

		Context("when there are no admin buildpacks in use", func() {
			It("cleans deleted admin buildpacks", func() {
				manager.Clean()

				Expect(utils.File_Exists(file_to_delete)).To(BeFalse())
				Expect(utils.File_Exists(file_to_keep)).To(BeTrue())
			})
		})

		Context("when an admin buildpack is in use", func() {
			var file_in_use string
			BeforeEach(func() {
				uri, _ := url.Parse("http://www.google.com")
				buildpacks_in_use = []StagingBuildpack{{uri, "efghi"}}

				file_in_use = path.Join(admin_buildpacks_dir, "efghi")
				create_populated_directory(file_in_use)
			})

			It("that buildpack doesn't get deleted", func() {
				manager.Clean()

				Expect(utils.File_Exists(file_to_delete)).To(BeFalse())
				Expect(utils.File_Exists(file_to_keep)).To(BeTrue())
				Expect(utils.File_Exists(file_in_use)).To(BeTrue())
			})
		})
	})

	Describe("list", func() {
		var system_buildpack string

		BeforeEach(func() {
			system_buildpack = path.Join(system_buildpacks_dir, "abcdef")
			create_populated_directory(system_buildpack)
		})

		Context("when there are multiple system buildpacks", func() {
			var additional_system_buildpacks []string

			BeforeEach(func() {
				additional_system_buildpacks = []string{
					path.Join(system_buildpacks_dir, "abc"),
					path.Join(system_buildpacks_dir, "def"),
				}
				for _, s := range additional_system_buildpacks {
					create_populated_directory(s)
				}
			})

			It("has a sorted list of system buildpacks", func() {
				buildpacks := additional_system_buildpacks
				buildpacks = append(buildpacks, system_buildpack)
				sort.Strings(buildpacks)
				Expect(manager.List()).To(Equal(buildpacks))
			})
		})

		Context("when there are admin buildpacks", func() {
			var admin_buildpack string

			BeforeEach(func() {

				url1, _ := url.Parse("http://example.com/buildpacks/uri/admin")
				url2, _ := url.Parse("http://example.com/buildpacks/uri/cant_find_admin")
				admin_buildpacks = []StagingBuildpack{
					StagingBuildpack{Url: url1, Key: "admin"},
					StagingBuildpack{Url: url2, Key: "cant_find_admin"},
				}

				admin_buildpack = path.Join(admin_buildpacks_dir, "admin")
				create_populated_directory(admin_buildpack)
			})

			It("has the correct number of buildpacks", func() {
				Expect(manager.List()).To(HaveLen(2))
			})

			It("copes with an admin buildpack not being there", func() {
				Expect(manager.List()).To(ContainElement(admin_buildpack))
			})

			It("includes the admin buildpacks which is there", func() {
				not_admin := path.Join(admin_buildpacks_dir, "cant_find_admin")
				Expect(manager.List()).ToNot(ContainElement(not_admin))
			})
		})

		Context("when there are no admin buildpacks", func() {
			It("includes the system buildpacks", func() {
				Expect(manager.List()).To(HaveLen(1))
				Expect(manager.List()).To(ContainElement(system_buildpack))
			})
		})
	})
})

func create_populated_directory(dir string) {
	os.MkdirAll(path.Join(dir, "a_buildpack_file"), 0755)
}
