package staging

import (
	cfg "dea/config"
	"dea/container"
	"dea/droplet"
	"dea/loggregator"
	"dea/task"
	thelpers "dea/testhelpers"
	temitter "dea/testhelpers/emitter"
	"dea/utils"
	"errors"
	"github.com/cloudfoundry/gordon"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

var _ = Describe("StagingTask", func() {
	var mockEmitter temitter.MockEmitter
	var config *cfg.Config
	var buildpacks_in_use []StagingBuildpack
	var attributes map[string]interface{}
	var staging StagingTask
	var mContainer *container.MockContainer
	var stgTask *stagingTask

	memory_limit_mb := uint64(256)
	disk_limit_mb := uint64(1025)

	BeforeEach(func() {
		mockEmitter = temitter.MockEmitter{}
		loggregator.SetStagingEmitter(&mockEmitter)

		base_dir, _ := ioutil.TempDir("", "staging_task_test")

		config = &cfg.Config{
			BaseDir:      base_dir,
			BuildpackDir: path.Join(base_dir, "buildpacks"),
			DirectoryServer: cfg.DirServerConfig{
				DeaPort: 1234,
			},
			Staging: cfg.StagingConfig{
				Environment:        map[string]string{"BUILDPACK_CACHE": "buildpack_cache_url"},
				MemoryLimitMB:      memory_limit_mb,
				DiskLimitMB:        disk_limit_mb,
				MaxStagingDuration: time.Duration(900) * time.Second,
			},
		}

		url1, _ := url.Parse("www.google.com")
		url2, _ := url.Parse("www.google2.com")
		buildpacks_in_use = []StagingBuildpack{
			{Key: "buildpack1", Url: url1},
			{Key: "buildpack2", Url: url2},
		}
		attributes = thelpers.Valid_staging_attributes()

		mContainer = &container.MockContainer{}
	})

	JustBeforeEach(func() {
		dropletRegistry := droplet.NewDropletRegistry(config.BaseDir)
		staging_message := NewStagingMessage(attributes)
		staging = NewStagingTask(config, staging_message, buildpacks_in_use,
			dropletRegistry, utils.Logger("staging_tasks_test_logger", nil))

		stgTask = staging.(*stagingTask)
		stgTask.Container = mContainer
	})

	AfterEach(func() {
		os.RemoveAll(config.BaseDir)
	})

	Describe("promise_stage", func() {
		It("assembles a shell command and initiates collection of task log", func() {
			stgTask.promise_stage()
			Expect(mContainer.MRunScript).To(ContainSubstring(`export FOO="BAR";`))
			Expect(mContainer.MRunScript).To(ContainSubstring(`export BUILDPACK_CACHE="buildpack_cache_url";`))
			Expect(mContainer.MRunScript).To(ContainSubstring(`export STAGING_TIMEOUT="900";`))
			Expect(mContainer.MRunScript).To(ContainSubstring(`export MEMORY_LIMIT="512m";`)) // the user assiged 512 should overwrite the system 256
			Expect(mContainer.MRunScript).To(ContainSubstring(`export VCAP_SERVICES="`))

			Expect(mContainer.MRunScript).To(MatchRegexp(".*/bin/run .*/plugin_config | tee -a"))
		})

		It("logs to the loggregator", func() {
			mContainer.MRunScriptStdout = "stdout message"
			mContainer.MRunScriptStderr = "stderr message"

			stgTask.promise_stage()

			app_id := staging.StagingMessage().App_id()
			Expect(mockEmitter.Messages).To(HaveLen(1))
			Expect(mockEmitter.ErrorMessages).To(HaveLen(1))
			Expect(mockEmitter.Messages[app_id][0]).To(Equal("stdout message"))
			Expect(mockEmitter.ErrorMessages[app_id][0]).To(Equal("stderr message"))
		})

		Context("when env variables need to be escaped", func() {
			BeforeEach(func() {
				startmsg := attributes["start_message"].(map[string]interface{})
				startmsg["env"] = []interface{}{"PATH=x y z", "FOO=z'y\"d", "BAR=", "BAZ=foo=baz"}
			})

			It("copes with spaces", func() {
				stgTask.promise_stage()
				Expect(mContainer.MRunScript).To(ContainSubstring(`export PATH="x y z";`))
			})

			It("copes with quotes", func() {
				stgTask.promise_stage()
				Expect(mContainer.MRunScript).To(ContainSubstring(`export FOO="z'y\"d";`))
			})

			It("copes with blanks", func() {
				stgTask.promise_stage()
				Expect(mContainer.MRunScript).To(ContainSubstring(`export BAR="";`))
			})

			It("copes with equal signs", func() {
				stgTask.promise_stage()
				Expect(mContainer.MRunScript).To(ContainSubstring(`export BAZ="foo=baz";`))
			})
		})

		Describe("timeouts", func() {
			BeforeEach(func() {
				config.Staging.MaxStagingDuration = time.Duration(500) * time.Millisecond
			})

			JustBeforeEach(func() {
				stgTask.staging_timeout_buffer = 0
			})

			Context("when the staging times out past the grace period", func() {
				It("fails with a Timeout Error", func() {
					mContainer.MRunScriptFunc = func() {
						time.Sleep(2 * time.Second)
					}

					err := stgTask.promise_stage()
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).To(Equal("Timed out"))
				})
			})

			Context("when the staging finishes within the grace period", func() {
				It("does not time out", func() {
					mContainer.MRunScriptFunc = func() {
						time.Sleep(250 * time.Millisecond)
					}

					err := stgTask.promise_stage()
					Expect(err).To(BeNil())
				})
			})
		})
	})

	Describe("task_info", func() {
		Context("when staging info file exists", func() {
			JustBeforeEach(func() {
				contents := `
---
detected_buildpack: Ruby/Rack
`
				staging_info := stgTask.workspace.staging_info_path()
				err := ioutil.WriteFile(staging_info, []byte(contents), 0755)
				Expect(err).To(BeNil())
			})

			It("parses staging info file", func() {
				Expect(stgTask.task_info()["detected_buildpack"]).To(Equal("Ruby/Rack"))
			})
		})

		Context("when staging info file does not exists", func() {
			It("preturns empty map", func() {
				Expect(stgTask.task_info()).To(HaveLen(0))
			})
		})
	})

	Describe("detected_buildpack", func() {
		JustBeforeEach(func() {
			contents := `
---
detected_buildpack: Ruby/Rack
`
			staging_info := stgTask.workspace.staging_info_path()
			err := ioutil.WriteFile(staging_info, []byte(contents), 0755)
			Expect(err).To(BeNil())
		})

		It("returns the detected buildpack", func() {
			Expect(stgTask.DetectedBuildpack()).To(Equal("Ruby/Rack"))
		})
	})

	Describe("path_in_container", func() {
		Context("when given path is not nil", func() {
			Context("when container path is set", func() {
				BeforeEach(func() {
					mContainer.MPath = "/container/path"
				})

				It("returns path inside warden container root file system", func() {
					Expect(staging.Path_in_container("path/to/file")).To(Equal("/container/path/tmp/rootfs/path/to/file"))
				})
			})
			Context("when container path is not set", func() {
				BeforeEach(func() {
					mContainer.MPath = ""
				})

				It("returns nil", func() {
					Expect(staging.Path_in_container("path/to/file")).To(Equal(""))
				})
			})
		})
		Context("when given path is nil", func() {
			Context("when container path is set", func() {
				BeforeEach(func() {
					mContainer.MPath = "/container/path"
				})

				It("returns path inside warden container root file system", func() {
					Expect(staging.Path_in_container("")).To(Equal("/container/path/tmp/rootfs/"))
				})
			})

			Context("when container path is not set", func() {
				BeforeEach(func() {
					mContainer.MPath = ""
				})

				It("returns nil", func() {
					Expect(staging.Path_in_container("")).To(Equal(""))
				})
			})
		})
	})

	Describe("start", func() {
		var mockPromises *mockStagingPromises
		BeforeEach(func() {
			mockPromises = newMockStagingPromises()
		})

		JustBeforeEach(func() {
			stgTask.StagingPromises = mockPromises
			stgTask.Task.TaskPromises = mockPromises
		})

		it_calls_callback := func(set_callback func(Callback), options map[string]error) {
			Describe("after_callback", func() {

				Context("when there is no callback registered", func() {
					It("doesn't not try to call registered callback", func() {
						staging.Start()
					})
				})

				Context("when there is callback registered", func() {
					var received_count int
					var received_error error

					JustBeforeEach(func() {
						received_count = 0
						received_error = nil
						set_callback(func(e error) error {
							received_count += 1
							received_error = e
							return nil
						})
					})

					Context("and staging task succeeds finishing callback", func() {
						It("calls registered callback without an error", func() {
							err := staging.Start()
							Expect(err).To(BeNil())
							Expect(received_count).To(Equal(1))
							Expect(received_error).To(BeNil())
						})
					})

					Context("and staging task fails before finishing callback", func() {
						JustBeforeEach(func() {
							mockPromises.errs = options
						})

						It("calls registered callback with an error", func() {
							err := staging.Start()
							Expect(err).ToNot(BeNil())
							Expect(received_count).To(Equal(1))
							Expect(received_error).ToNot(BeNil())
							Expect(received_error.Error()).To(Equal("failing promise"))
						})
					})

					Context("and the callback itself fails", func() {
						JustBeforeEach(func() {
							received_count = 0
							received_error = nil
							set_callback(func(e error) error {
								received_count += 1
								return errors.New("failing callback")
							})
						})

						It("cleans up workspace", func() {
							staging.Start()
							Expect(utils.File_Exists(stgTask.workspace.Workspace_dir())).To(BeFalse())
						})

						It("calls registered callback exactly once", func() {
							staging.Start()
							Expect(received_count).To(Equal(1))
						})

						Context("and there is no error from staging", func() {
							It("raises error raised in the callback", func() {
								err := staging.Start()
								Expect(err).ToNot(BeNil())
								Expect(err.Error()).To(Equal("failing callback"))
							})
						})

						Context("and there is an error from staging", func() {
							JustBeforeEach(func() {
								mockPromises.errs = options
							})

							It("raises the staging error", func() {
								err := staging.Start()
								Expect(err.Error()).To(Equal("failing callback"))
							})
						})
					})
				})
			})
		}

		Context("setup callback", func() {
			it_calls_callback(func(c Callback) { staging.SetAfter_setup_callback(c) }, map[string]error{"promise_app_download": errors.New("failing promise")})
		})

		Context("complete callback", func() {
			it_calls_callback(func(c Callback) { staging.SetAfter_complete_callback(c) }, map[string]error{"promise_stage": errors.New("failing promise")})
		})

		Context("when a script fails", func() {
			BeforeEach(func() {
				mockPromises.errs["promise_stage"] = errors.New("Script Failed")
			})

			It("still copies out the task log", func() {
				staging.Start()
				Expect(mockPromises.invokeCount["promise_task_log"]).To(Equal(1))
			})

			It("propagates the error", func() {
				err := staging.Start()
				Expect(err.Error()).To(Equal("Script Failed"))
			})

			It("returns an error in response", func() {
				var response error
				staging.SetAfter_upload_callback(func(e error) error {
					response = e
					return nil
				})
				staging.Start()

				Expect(response.Error()).To(Equal("Script Failed"))
			})

			It("does not uploads droplet", func() {
				staging.Start()

				//should not invoke resolve_staging_upload
				Expect(mockPromises.invokeCount["promise_app_upload"]).To(Equal(0))
			})
		})

		Describe("bind_mounts", func() {
			It("includes the workspace dir", func() {
				d := stgTask.workspace.Workspace_dir()
				Expect(stgTask.bind_mounts()).To(ContainElement(&warden.CreateRequest_BindMount{
					SrcPath: &d,
					DstPath: &d,
				}))
			})

			It("includes the build pack url", func() {
				d := stgTask.workspace.buildpack_dir()
				Expect(stgTask.bind_mounts()).To(ContainElement(&warden.CreateRequest_BindMount{
					SrcPath: &d,
					DstPath: &d,
				}))
			})

			It("includes the configured bind mounts", func() {
				a := "a"
				b := "b"
				stgTask.bindMounts = []map[string]string{{
					"src_path": a,
					"dst_path": b,
				}}

				Expect(stgTask.bind_mounts()).To(ContainElement(&warden.CreateRequest_BindMount{
					SrcPath: &a,
					DstPath: &b,
				}))
			})
		})

		Context("when buildpack_cache_download_uri is provided", func() {
			BeforeEach(func() {
				attributes["buildpack_cache_download_uri"] = "http://www.someurl.com"
			})

			It("downloads buildpack cache", func() {
				staging.Start()
				Expect(mockPromises.invokeCount["promise_buildpack_cache_download"]).To(Equal(1))
			})
		})

		It("performs staging operations in correct order", func() {
			order := []string{
				"unpack_app",
				"unpack_buildpack_cache",
				"stage",
				"pack_app",
				"copy_out",
				"save_droplet",
				"log_upload_started",
				"staging_info",
				"task_log",
			}

			staging.Start()

			for i := 1; i < len(order); i++ {
				Expect(mockPromises.order["promise_"+order[i-1]]).To(BeNumerically("<", mockPromises.order["promise_"+order[i]]))
			}
		})

		It("performs staging upload operations in correct order", func() {
			order := []string{
				"promise_app_upload",
				"promise_save_buildpack_cache",
				"Promise_destroy",
			}

			staging.Start()

			for i := 1; i < len(order); i++ {
				Expect(mockPromises.order[order[i-1]]).To(BeNumerically("<", mockPromises.order[order[i]]))
			}
		})

		It("triggers callbacks in correct order", func() {
			complete := 9999
			upload := -1
			staging.SetAfter_complete_callback(func(e error) error {
				complete = mockPromises.step
				return e
			})
			staging.SetAfter_upload_callback(func(e error) error {
				upload = mockPromises.step
				return e
			})

			staging.Start()
			Expect(complete).To(BeNumerically("<", upload))
			Expect(complete).To(BeNumerically("<=", mockPromises.order["promise_app_upload"]))
			Expect(upload).To(BeNumerically(">=", mockPromises.order["promise_save_buildpack_cache"]))
		})

		Context("when the upload fails", func() {

			it_raises_and_returns_an_error := func() {
				var response error
				staging.SetAfter_upload_callback(func(e error) error {
					response = e
					return nil
				})

				err := staging.Start()

				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal("error"))
				Expect(response.Error()).To(Equal("error"))
			}

			It("copes with uploading errors", func() {
				mockPromises.errs["promise_app_upload"] = errors.New("error")

				it_raises_and_returns_an_error()
			})

			It("copes with buildpack cache errors", func() {
				mockPromises.errs["promise_save_buildpack_cache"] = errors.New("error")

				it_raises_and_returns_an_error()
			})
		})
	})

	Describe("Stop", func() {
		var mockPromises *mockStagingPromises
		BeforeEach(func() {
			mockPromises = newMockStagingPromises()
		})

		JustBeforeEach(func() {
			stgTask.StagingPromises = mockPromises
			stgTask.Task.TaskPromises = mockPromises
		})

		Context("if container exists", func() {
			BeforeEach(func() {
				mContainer.MHandle = "maria"
			})

			It("sends stop request to warden container", func() {
				staging.Stop()

				Expect(mockPromises.invokeCount["Promise_stop"]).To(Equal(1))
			})
		})

		Context("if container does not exist", func() {
			BeforeEach(func() {
				mContainer.MHandle = ""
			})

			It("does NOT send stop request to warden container", func() {
				staging.Stop()
				Expect(mockPromises.invokeCount["Promise_stop"]).To(Equal(0))
			})
		})

		It("triggers after stop callback", func() {
			invoked := false
			staging.SetAfter_stop_callback(func(e error) error {
				invoked = true
				return e
			})
			staging.Stop()
			Expect(invoked).To(BeTrue())
		})

		It("unregisters after complete callback", func() {
			// Emulate staging stop while running staging
			staging.SetAfter_setup_callback(func(e error) error {
				staging.Stop()
				return e
			})
			invoked := false
			staging.SetAfter_complete_callback(func(e error) error {
				invoked = true
				return e
			})

			staging.Start()
			Expect(invoked).To(BeFalse())
		})
	})

	Describe("MemoryLimit", func() {
		It("exports memory in bytes as specified in the config file", func() {
			Expect(staging.MemoryLimit()).To(Equal(cfg.Mebi * cfg.Memory(memory_limit_mb)))
		})

		Context("when unspecified", func() {
			BeforeEach(func() {
				config.Staging.MemoryLimitMB = 0
			})

			It("uses 1GB as a default", func() {
				Expect(staging.MemoryLimit()).To(Equal(cfg.Mebi * 1024))
			})
		})
	})

	Describe("DiskLimit", func() {
		It("exports disk in bytes as specified in the config file", func() {
			Expect(staging.DiskLimit()).To(Equal(cfg.MB * cfg.Disk(disk_limit_mb)))
		})

		Context("when unspecified", func() {
			BeforeEach(func() {
				config.Staging.DiskLimitMB = 0
			})

			It("uses 2GB as a default", func() {
				Expect(staging.DiskLimit()).To(Equal(cfg.MB * 2 * 1024))
			})
		})
	})

	Describe("promise_prepare_staging_log", func() {
		It("assembles a shell command that creates staging_task.log file for tailing it", func() {
			stgTask.promise_prepare_staging_log()
			Expect(mContainer.MRunScript).To(Equal("mkdir -p /tmp/staged/logs && touch /tmp/staged/logs/staging_task.log"))
		})
	})

	Describe("promise_app_download", func() {
		var staging_app_file_path string
		JustBeforeEach(func() {
			staging_app_file_path = path.Join(stgTask.workspace.Workspace_dir(), "app.zip")
		})

		Context("when there is an error", func() {
			It("expects an error", func() {
				err := stgTask.promise_app_download()
				Expect(err).ToNot(BeNil())
			})

			It("should not create an app file", func() {
				stgTask.promise_app_download()
				Expect(utils.File_Exists(stgTask.workspace.downloaded_app_package_path())).To(BeFalse())
			})
		})

		Context("when there is no error", func() {
			var httpserver *httptest.Server
			BeforeEach(func() {
				httpserver = thelpers.NewFileServer()
				attributes["download_uri"] = httpserver.URL + "/download"
			})
			AfterEach(func() {
				httpserver.Close()
			})

			It("should rename the file", func() {
				stgTask.promise_app_download()
				finfo, err := os.Stat(staging_app_file_path)
				Expect(err).To(BeNil())

				Expect(os.IsNotExist(err)).To(BeFalse())
				Expect(finfo.Mode().Perm()).To(Equal(os.FileMode(0744)))
			})
		})
	})

	Describe("promise_buildpack_cache_download", func() {
		var buildpack_cache_dest string
		JustBeforeEach(func() {
			buildpack_cache_dest = path.Join(stgTask.workspace.Workspace_dir(), "buildpack_cache.tgz")
		})

		Context("when there is an error", func() {
			It("expects an error", func() {
				err := stgTask.promise_buildpack_cache_download()
				Expect(err).ToNot(BeNil())
			})

			It("should not create an app file", func() {
				stgTask.promise_buildpack_cache_download()
				Expect(utils.File_Exists(buildpack_cache_dest)).To(BeFalse())
			})
		})

		Context("when there is no error", func() {
			var httpserver *httptest.Server
			BeforeEach(func() {
				httpserver = thelpers.NewFileServer()
				attributes["buildpack_cache_download_uri"] = httpserver.URL + "/download"
			})
			AfterEach(func() {
				httpserver.Close()
			})

			It("should rename the file", func() {
				stgTask.promise_buildpack_cache_download()
				finfo, err := os.Stat(buildpack_cache_dest)
				Expect(err).To(BeNil())

				Expect(os.IsNotExist(err)).To(BeFalse())
				Expect(finfo.Mode().Perm()).To(Equal(os.FileMode(0744)))
			})
		})
	})

	Describe("promise_unpack_app", func() {
		It("assembles a shell command", func() {
			stgTask.promise_unpack_app()
			workspace_dir := stgTask.workspace.Workspace_dir()
			Expect(mContainer.MRunScript).To(ContainSubstring("unzip -q " + workspace_dir + "/app.zip -d /tmp/unstaged"))
		})

		It("logs to the loggregator", func() {
			mContainer.MRunScriptStdout = "stdout message"
			mContainer.MRunScriptStderr = "stderr message"

			stgTask.promise_unpack_app()

			app_id := staging.StagingMessage().App_id()
			Expect(mockEmitter.Messages).To(HaveLen(1))
			Expect(mockEmitter.ErrorMessages).To(HaveLen(1))
			Expect(mockEmitter.Messages[app_id][0]).To(Equal("stdout message"))
			Expect(mockEmitter.ErrorMessages[app_id][0]).To(Equal("stderr message"))
		})
	})

	Describe("promise_unpack_buildpack_cache", func() {
		Context("when buildpack cache does not exist", func() {
			It("does not run a warden command", func() {
				stgTask.promise_unpack_buildpack_cache()
				Expect(mContainer.MRunScript).To(Equal(""))
			})
		})

		Context("when buildpack cache exists", func() {
			JustBeforeEach(func() {
				f, _ := os.Create(path.Join(stgTask.workspace.Workspace_dir(), "buildpack_cache.tgz"))
				f.Close()
			})

			It("assembles a shell command", func() {
				stgTask.promise_unpack_app()
				workspace_dir := stgTask.workspace.Workspace_dir()
				Expect(mContainer.MRunScript).To(ContainSubstring("unzip -q " + workspace_dir + "/app.zip -d /tmp/unstaged"))
			})

			It("logs to the loggregator", func() {
				mContainer.MRunScriptStdout = "stdout message"
				mContainer.MRunScriptStderr = "stderr message"

				stgTask.promise_unpack_app()

				app_id := staging.StagingMessage().App_id()
				Expect(mockEmitter.Messages).To(HaveLen(1))
				Expect(mockEmitter.ErrorMessages).To(HaveLen(1))
				Expect(mockEmitter.Messages[app_id][0]).To(Equal("stdout message"))
				Expect(mockEmitter.ErrorMessages[app_id][0]).To(Equal("stderr message"))
			})
		})
	})

	Describe("promise_pack_app", func() {
		It("assembles a shell command", func() {
			stgTask.promise_pack_app()
			Expect(mContainer.MRunScript).To(ContainSubstring("cd /tmp/staged && COPYFILE_DISABLE=true tar -czf /tmp/droplet.tgz ."))
		})
	})

	Describe("promise_save_buildpack_cache", func() {
		var mockPromises *mockStagingPromises
		BeforeEach(func() {
			mockPromises = newMockStagingPromises()
		})

		JustBeforeEach(func() {
			mockPromises.stagingPromises = stgTask.StagingPromises
			stgTask.StagingPromises = mockPromises
			stgTask.Task.TaskPromises = mockPromises
		})

		Context("when packing succeeds", func() {
			BeforeEach(func() {
				mockPromises.invokePromise["promise_save_buildpack_cache"] = true
			})

			It("copies out the buildpack cache", func() {
				stgTask.promise_save_buildpack_cache()
				Expect(mockPromises.invokeCount["promise_copy_out_buildpack_cache"]).To(Equal(1))
			})

			It("uploads the buildpack cache", func() {
				stgTask.promise_save_buildpack_cache()
				Expect(mockPromises.invokeCount["promise_buildpack_cache_upload"]).To(Equal(1))
			})
		})

		Context("when packing fails", func() {
			BeforeEach(func() {
				mockPromises.invokePromise["promise_save_buildpack_cache"] = true
				mockPromises.errs["promise_pack_buildpack_cache"] = errors.New("fail")
			})

			It("does not copy out the buildpack cache", func() {
				err := stgTask.promise_save_buildpack_cache()
				Expect(err).ToNot(BeNil())
				Expect(mockPromises.invokeCount["promise_copy_out_buildpack_cache"]).To(Equal(0))
			})

			It("does not upload the buildpack cache", func() {
				err := stgTask.promise_save_buildpack_cache()
				Expect(err).ToNot(BeNil())
				Expect(mockPromises.invokeCount["promise_buildpack_cache_upload"]).To(Equal(0))
			})
		})
	})

	Describe("promise_app_upload", func() {
		Context("when there is an error", func() {
			It("expects an error", func() {
				err := stgTask.promise_app_upload()
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when there is no error", func() {
			var httpserver *httptest.Server
			BeforeEach(func() {
				httpserver = thelpers.NewFileServer()
				attributes["upload_uri"] = httpserver.URL + "/upload"
			})
			AfterEach(func() {
				httpserver.Close()
			})

			JustBeforeEach(func() {
				droplet := stgTask.workspace.staged_droplet_path()
				os.MkdirAll(path.Dir(droplet), 0755)
				f, _ := os.Create(droplet)
				f.Close()
			})

			It("should rename the file", func() {
				err := stgTask.promise_app_upload()
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("promise_buildpack_cache_upload", func() {
		Context("when there is an error", func() {
			It("expects an error", func() {
				err := stgTask.promise_buildpack_cache_upload()
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when there is no error", func() {
			var httpserver *httptest.Server
			BeforeEach(func() {
				httpserver = thelpers.NewFileServer()
				attributes["buildpack_cache_upload_uri"] = httpserver.URL + "/upload"
			})
			AfterEach(func() {
				httpserver.Close()
			})

			JustBeforeEach(func() {
				bpcache := stgTask.workspace.staged_buildpack_cache_path()
				os.MkdirAll(path.Dir(bpcache), 0755)
				f, _ := os.Create(bpcache)
				f.Close()
			})

			It("should rename the file", func() {
				err := stgTask.promise_buildpack_cache_upload()
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("promise_copy_out", func() {
		It("should send copying out request", func() {
			stgTask.promise_copy_out()
			Expect(mContainer.MCopyOutSrc).To(Equal("/tmp/droplet.tgz"))
			Expect(mContainer.MCopyOutDest).To(Equal(stgTask.workspace.staged_droplet_dir()))
		})
	})

	Describe("promise_save_droplet", func() {
		var sha1 string
		JustBeforeEach(func() {
			dropletPath := stgTask.workspace.staged_droplet_path()
			os.MkdirAll(path.Dir(dropletPath), 0755)
			ioutil.WriteFile(dropletPath, []byte("test"), 0755)
			s, _ := utils.SHA1Digest(dropletPath)
			sha1 = string(s)
			stgTask.dropletRegistry.Put(sha1)
		})

		It("saves droplet and droplet sha", func() {
			stgTask.promise_save_droplet()
			Expect(staging.DropletSHA1()).To(Equal(sha1))
		})
	})

	Describe("promise_copy_out_buildpack_cache", func() {
		It("should send copying out request", func() {
			stgTask.promise_copy_out_buildpack_cache()
			Expect(mContainer.MCopyOutSrc).To(Equal("/tmp/buildpack_cache.tgz"))
			Expect(mContainer.MCopyOutDest).To(Equal(stgTask.workspace.staged_droplet_dir()))
		})
	})

	Describe("promise_task_log", func() {
		It("should send copying out request", func() {
			stgTask.promise_task_log()
			Expect(mContainer.MCopyOutSrc).To(Equal("/tmp/staged/logs/staging_task.log"))
			Expect(mContainer.MCopyOutDest).To(Equal(stgTask.workspace.Workspace_dir()))
		})
	})

	Describe("promise_staging_info", func() {
		It("should send copying out request", func() {
			stgTask.promise_staging_info()
			Expect(mContainer.MCopyOutSrc).To(Equal("/tmp/staged/staging_info.yml"))
			Expect(mContainer.MCopyOutDest).To(Equal(stgTask.workspace.Workspace_dir()))
		})

	})
})

type mockStagingPromises struct {
	errs        map[string]error
	invokeCount map[string]int
	step        int
	order       map[string]int

	stagingPromises StagingPromises
	taskPromises    task.TaskPromises
	invokePromise   map[string]bool
}

func newMockStagingPromises() *mockStagingPromises {
	return &mockStagingPromises{
		errs:        make(map[string]error),
		invokeCount: make(map[string]int),
		order:       make(map[string]int),

		invokePromise: make(map[string]bool),
	}
}

func (m *mockStagingPromises) promise_app_download() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_buildpack_cache_download() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_limit_disk() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_limit_memory() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_prepare_staging_log() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_app_dir() error {
	m.inc()
	return m.err()
}

func (m *mockStagingPromises) promise_unpack_app() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_unpack_buildpack_cache() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_stage() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_pack_app() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_copy_out() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_save_droplet() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_log_upload_started() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_app_upload() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_pack_buildpack_cache() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_copy_out_buildpack_cache() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_buildpack_cache_upload() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_staging_info() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) promise_task_log() error {
	m.inc()
	return m.err()
}

func (m *mockStagingPromises) promise_save_buildpack_cache() error {
	m.inc()
	if m.invokePromise["promise_save_buildpack_cache"] && m.stagingPromises != nil {
		return m.stagingPromises.promise_save_buildpack_cache()
	}

	return m.err()
}

func (m *mockStagingPromises) Promise_stop() error {
	m.inc()
	return m.err()
}
func (m *mockStagingPromises) Promise_destroy() {
	m.inc()
}

func (m *mockStagingPromises) err() error {
	return m.errs[getMethodName()]
}

func (m *mockStagingPromises) inc() {
	methodName := getMethodName()
	m.invokeCount[methodName]++

	m.order[methodName] = m.step
	m.step++
}

func getMethodName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(3, pc)
	f := runtime.FuncForPC(pc[0])
	dot := strings.LastIndex(f.Name(), ".")
	return f.Name()[dot+1:]
}
