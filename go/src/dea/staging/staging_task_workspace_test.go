package staging

import (
	"dea"
	thelpers "dea/testhelpers"
	"dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
)

var _ = Describe("StagingTaskWorkspace", func() {
	var basedir string
	var admin_buildpacks_dir string
	var system_buildpacks_dir string
	var stagingMsg *StagingMessage
	var subject StagingTaskWorkspace
	var admin_buildpacks []map[string]interface{}

	env_properties := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	buildpacks_in_use := []dea.StagingBuildpack{}

	BeforeEach(func() {
		basedir, _ = ioutil.TempDir("", "staging_task_workspace")
		os.MkdirAll(basedir, 0755)

		admin_buildpacks_dir = path.Join(basedir, "admin_buildpacks")

		_, caller, _, _ := runtime.Caller(0)
		system_buildpacks_dir = path.Join(caller, "../../../../../buildpacks/vendor")

		admin_buildpacks = []map[string]interface{}{
			{"url": "http://www.example.com/buildpacks/uri/abcdef", "key": "abcdef"},
			{"url": "http://www.example.com/buildpacks/uri/ghijk", "key": "ghijk"},
		}

	})

	AfterEach(func() {
		os.RemoveAll(basedir)
	})

	JustBeforeEach(func() {
		staging_message := map[string]interface{}{
			"admin_buildpacks": admin_buildpacks,
			"properties":       env_properties,
		}
		stagingMsg = NewStagingMessage(staging_message)
		subject = NewStagingTaskWorkspace(basedir, system_buildpacks_dir, stagingMsg, buildpacks_in_use)
	})

	Describe("preparing the workspace", func() {
		var httpServer *httptest.Server

		BeforeEach(func() {
			httpServer = thelpers.NewFileServer()

			admin_buildpacks[0]["url"] = httpServer.URL + "/buildpacks/uri/abcdef"
			admin_buildpacks[1]["url"] = httpServer.URL + "/buildpacks/uri/ghijk"
		})
		AfterEach(func() {
			httpServer.Close()
		})

		It("downloads the admin buildpacks", func() {
			subject.Prepare()

			expected_file_name := path.Join(admin_buildpacks_dir, "abcdef")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())
			expected_file_name = path.Join(admin_buildpacks_dir, "ghijk")
			Expect(utils.File_Exists(expected_file_name)).To(BeTrue())
		})

		Describe("the plugin config file", func() {
			Context("when admin buildpack exists", func() {
				var admin_buildpack string
				var plugin_config map[string]interface{}

				JustBeforeEach(func() {
					admin_buildpack = path.Join(subject.Admin_buildpacks_dir(), "abcdef")
					os.MkdirAll(admin_buildpack, 0755)

					subject.Prepare()
					plugin_config = make(map[string]interface{})

					err := utils.Yaml_Load(subject.Plugin_config_path(), &plugin_config)
					Expect(err).To(BeNil())
				})

				It("includes the admin buildpacks", func() {
					Expect(plugin_config["buildpack_dirs"]).To(ContainElement(admin_buildpack))
				})

				It("admin buildpack should come first", func() {
					bpdirs := plugin_config["buildpack_dirs"].([]interface{})
					Expect(bpdirs[0]).To(Equal(admin_buildpack))
				})
			})

			Context("when multiple admin buildpacks exist", func() {
				var admin_buildpack string
				var another_buildpack string
				var plugin_config map[string]interface{}

				JustBeforeEach(func() {
					admin_buildpack = path.Join(subject.Admin_buildpacks_dir(), "abcdef")
					os.MkdirAll(admin_buildpack, 0755)

					another_buildpack = path.Join(subject.Admin_buildpacks_dir(), "xyz")
					os.MkdirAll(admin_buildpack, 0755)

					subject.Prepare()
					plugin_config = make(map[string]interface{})

					err := utils.Yaml_Load(subject.Plugin_config_path(), &plugin_config)
					Expect(err).To(BeNil())
				})

				It("only returns buildpacks specified in start message", func() {
					Expect(plugin_config["buildpack_dirs"]).To(ContainElement(admin_buildpack))
					Expect(plugin_config["buildpack_dirs"]).ToNot(ContainElement(another_buildpack))
				})
			})

			Context("when the config lists multiple admin buildpacks which exist on disk", func() {
				var admin_buildpack string
				var another_buildpack string
				var plugin_config map[string]interface{}

				JustBeforeEach(func() {
					admin_buildpack = path.Join(subject.Admin_buildpacks_dir(), "abcdef")
					os.MkdirAll(admin_buildpack, 0755)

					another_buildpack = path.Join(subject.Admin_buildpacks_dir(), "ghijk")
					os.MkdirAll(admin_buildpack, 0755)

					subject.Prepare()
					plugin_config = make(map[string]interface{})

					err := utils.Yaml_Load(subject.Plugin_config_path(), &plugin_config)
					Expect(err).To(BeNil())
				})

				Context("when the buildpacks are ordered admin_buildpack, another_buildpack", func() {
					It("returns the buildpacks in the order of the admin_buildpacks message", func() {
						bpdirs := plugin_config["buildpack_dirs"].([]interface{})
						Expect(bpdirs[0]).To(Equal(admin_buildpack))
						Expect(bpdirs[1]).To(Equal(another_buildpack))
					})
				})

				Context("when the buildpacks are ordered another_buildpack, admin_buildpack", func() {
					BeforeEach(func() {
						admin_buildpacks[0], admin_buildpacks[1] = admin_buildpacks[1], admin_buildpacks[0]
					})

					It("returns the buildpacks in the order of the admin_buildpacks message", func() {
						bpdirs := plugin_config["buildpack_dirs"].([]interface{})
						Expect(bpdirs[0]).To(Equal(another_buildpack))
						Expect(bpdirs[1]).To(Equal(admin_buildpack))
					})
				})
			})

			Context("when admin buildpack does not exist", func() {
				var plugin_config map[string]interface{}

				BeforeEach(func() {
					admin_buildpacks = []map[string]interface{}{}
				})

				JustBeforeEach(func() {
					subject.Prepare()
					plugin_config = make(map[string]interface{})

					err := utils.Yaml_Load(subject.Plugin_config_path(), &plugin_config)
					Expect(err).To(BeNil())
				})

				It("has the right source, destination and cache directories", func() {
					Expect(plugin_config["source_dir"]).To(Equal("/tmp/unstaged"))
					Expect(plugin_config["dest_dir"]).To(Equal("/tmp/staged"))
					Expect(plugin_config["cache_dir"]).To(Equal("/tmp/cache"))
				})

				It("includes the specified environment config", func() {
					Expect(plugin_config["environment"]).To(HaveLen(len(env_properties)))
					env := plugin_config["environment"].(map[interface{}]interface{})
					for k, v := range env_properties {
						Expect(env[k]).To(Equal(v))
					}
				})

				It("includes the staging info path", func() {
					Expect(plugin_config["staging_info_name"]).To(Equal("staging_info.yml"))
				})

				It("include the system buildpacks", func() {
					files, err := ioutil.ReadDir(system_buildpacks_dir)
					Expect(err).To(BeNil())
					bpdirs := make([]interface{}, 0, len(files))
					for _, f := range files {
						bpdirs = append(bpdirs, path.Join(system_buildpacks_dir, f.Name()))
					}

					Expect(plugin_config["buildpack_dirs"]).To(Equal(bpdirs))
				})
			})
		})
	})

	It("creates the tmp folder", func() {
		Expect(utils.File_Exists(subject.Tmp_dir())).To(BeTrue())
	})

	It("creates the workspace dir folder", func() {
		Expect(utils.File_Exists(subject.Workspace_dir())).To(BeTrue())
	})
})
