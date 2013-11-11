package env_test

import (
	cfg "dea/config"
	. "dea/env"
	"dea/staging"
	"dea/starting"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

var _ = Describe("Env", func() {
	var services []map[string]interface{}
	var environment []string
	var start_message map[string]interface{}
	var instance *starting.Instance
	var exported_variables string

	BeforeEach(func() {
		service := map[string]interface{}{
			"name":        "elephantsql-vip-uat",
			"label":       "elephantsql-n/a",
			"credentials": map[string]interface{}{"uri": "postgres://user:pass@host:5432/db"},
			"plan":        "panda",
			"tags":        []string{"elephantsql", "mysql"},
		}
		services = []map[string]interface{}{service}
		environment = []string{"A=one_value", "B=with spaces", "C=with'quotes\"double", "D=referencing $A", "E=with=equals", "F="}
		start_message = map[string]interface{}{
			"index":               float64(1),
			"application_id":      "ab-cd-ef",
			"instance_id":         "451f045fd16427bb99c895a2649b7b2a",
			"application_name":    "vip-uat-sidekiq",
			"application_uris":    []string{"first_uri", "second_uri"},
			"application_version": "fake-version-no",
			"droplet_sha1":        "abcdef",
			"droplet_uri":         "http://example.com/droplet_uri",
			"limits": map[string]interface{}{
				"mem":  float64(512),
				"disk": float64(1024),
				"fds":  float64(16384)},
			"cc_partition":             "default",
			"instance_index":           float64(0),
			"warden_handle":            "1234",
			"instance_host_port":       float64(2345),
			"instance_container_port":  float64(4567),
			"state_STARTING_timestamp": time.Now(),
		}
	})

	JustBeforeEach(func() {
		start_message["env"] = environment
		start_message["services"] = services

		config := cfg.Config{}
		instance = starting.NewInstance(start_message, &config, nil, "127.0.0.1")
		Expect(instance).ToNot(BeNil())
	})

	it_exports := func(name, value string) {
		It("exports name as value", func() {
			Expect(exported_variables).To(MatchRegexp("export %s=\".*%s.*\"", name, value))
		})
	}

	it_does_not_export := func(name string) {
		It("does not export name", func() {
			match, err := regexp.MatchString("export "+name, exported_variables)
			Expect(err).To(BeNil())
			Expect(match).To(BeFalse())

		})
	}

	Context("when running from the starting (instance) task", func() {
		var subject *Env

		JustBeforeEach(func() {
			subject = NewEnv(starting.NewRunningEnv(instance))
		})

		Describe("exported_system_environment_variables", func() {
			JustBeforeEach(func() {
				exported_variables, _ = subject.ExportedSystemEnvironmentVariables()
			})

			it_exports("VCAP_APPLICATION", `"instance_index\\":0`)
			it_exports("VCAP_SERVICES", `"plan\\":\\"panda\\"`)
			it_exports("VCAP_APP_HOST", "0.0.0.0")
			it_exports("VCAP_APP_PORT", "4567")
			it_exports("PORT", `\$VCAP_APP_PORT`)
			it_exports("MEMORY_LIMIT", "512m")
			it_exports("HOME", `\$PWD/app`)
			it_exports("TMPDIR", `\$PWD/tmp`)

			Context("when it has a DB", func() {
				it_exports("DATABASE_URL", "postgres://user:pass@host:5432/db")
			})

			Context("when it does NOT have a DB", func() {
				BeforeEach(func() {
					services = []map[string]interface{}{}
				})
				it_does_not_export("DATABASE_URL")
			})
		})

		Describe("exported_user_environment_variables", func() {
			JustBeforeEach(func() {
				exported_variables = subject.ExportedUserEnvironmentVariables()
			})

			it_exports("A", "one_value")
			it_exports("B", "with spaces")
			it_exports("C", `with'quotes\\"double`)
			it_exports("D", "referencing \\$A")
			it_exports("E", "with=equals")
			it_exports("F", "")
		})

	})

	Context("when running from the staging task", func() {
		var baseDir string
		var staging_message map[string]interface{}
		var subject *Env

		BeforeEach(func() {
			staging_message = map[string]interface{}{
				"app_id":  "fake-app-id",
				"task_id": "fake-task-id",
				"properties": map[string]interface{}{
					"services":  services,
					"buildpack": nil,
					"resources": map[string]float64{
						"memory": 512,
						"disk":   1024,
						"fds":    16384,
					},
					"environment": environment,
					"meta": map[string]interface{}{
						"command": "some_command",
					},
				},
				"download_uri":                 "https://download_uri",
				"upload_uri":                   "http://upload_uri",
				"buildpack_cache_download_uri": "https://buildpack_cache_download_uri",
				"buildpack_cache_upload_uri":   "http://buildpack_cache_upload_uri",
				"start_message":                start_message,
			}
		})

		JustBeforeEach(func() {
			baseDir, _ = ioutil.TempDir("", "env_test")
			config := &cfg.Config{BaseDir: baseDir}
			config.Staging.MaxStagingDuration = 900
			env := make(map[string]string)
			env["BUILDPACK_CACHE"] = ""
			config.Staging.Environment = env

			staging_message := staging.NewStagingMessage(staging_message)
			staging_task := staging.NewStagingTask(config, staging_message,
				[]staging.StagingBuildpack{}, nil, nil)
			subject = NewEnv(staging.NewStagingEnv(staging_task))
		})

		AfterEach(func() {
			os.RemoveAll(baseDir)
		})

		Describe("exported_system_environment_variables", func() {
			JustBeforeEach(func() {
				exported_variables, _ = subject.ExportedSystemEnvironmentVariables()
			})

			it_exports("VCAP_APPLICATION", `"mem\\":512`)
			it_exports("VCAP_SERVICES", `"plan\\":\\"panda\\"`)
			it_exports("MEMORY_LIMIT", "512m")

			Context("when it has a DB", func() {
				it_exports("DATABASE_URL", "postgres://user:pass@host:5432/db")
			})

			Context("when it does NOT have a DB", func() {
				BeforeEach(func() {
					services = []map[string]interface{}{}
				})
				it_does_not_export("DATABASE_URL")
			})
		})

		Describe("exported_user_environment_variables", func() {
			JustBeforeEach(func() {
				exported_variables = subject.ExportedUserEnvironmentVariables()
			})

			it_exports("A", "one_value")
			it_exports("B", "with spaces")
			it_exports("C", `with'quotes\\"double`)
			it_exports("D", "referencing \\$A")
			it_exports("E", "with=equals")
			it_exports("F", "")
		})

		Describe("exported_environment_variables", func() {
			BeforeEach(func() {
				environment = []string{"PORT=stupid idea"}
			})

			JustBeforeEach(func() {
				exported_variables, _ = subject.ExportedEnvironmentVariables()
			})

			it_exports("PORT", "stupid idea")
		})
	})
})
