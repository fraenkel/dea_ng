package env_test

import (
	"dea"
	cfg "dea/config"
	. "dea/env"
	"dea/staging"
	"dea/starting"
	"dea/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"regexp"
	"time"
)

var _ = Describe("Env", func() {
	var service map[string]interface{}
	var services []map[string]interface{}
	var environment []string
	var start_message map[string]interface{}
	var instance *starting.Instance
	var exported_variables string

	BeforeEach(func() {
		service = map[string]interface{}{
			"name":             "elephantsql-vip-uat",
			"label":            "elephantsql-n/a",
			"provider":         "elephantsql",
			"version":          "n/a",
			"vendor":           "elephantsql",
			"options":          map[string]interface{}{},
			"credentials":      map[string]interface{}{"uri": "postgres://user:pass@host:5432/db"},
			"plan":             "panda",
			"plan_option":      "plan_option",
			"tags":             []string{"elephantsql", "mysql"},
			"syslog_drain_url": "syslog://drain-url.example.com:514",
		}
		services = []map[string]interface{}{service}
		environment = []string{"A=one_value", "B=with spaces", "C=with'quotes\"double", "D=referencing $A", "E=with=equals", "F="}
		start_message = testhelpers.Valid_instance_attributes(false)
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

		Describe("vcap_application", func() {
			var vcap_app map[string]interface{}

			JustBeforeEach(func() {
				vcap_app = subject.VcapApplication()
			})

			includesKey := func(key string) {
				It("includes "+key, func() {
					Expect(vcap_app).To(HaveKey(key))
				})
			}

			includesKey("instance_id")
			includesKey("instance_index")
			includesKey("application_version")
			includesKey("application_name")
			includesKey("application_uris")

			It("includes the time the instance was started", func() {
				var i int64
				Expect(vcap_app["started_at"]).Should(BeAssignableToTypeOf(time.Now()))
				Expect(vcap_app["started_at_timestamp"]).Should(BeAssignableToTypeOf(i))
			})

			It("includes the host and port the instance should listen on", func() {
				Expect(vcap_app["host"]).Should(Equal(starting.HOST))
				Expect(vcap_app["port"]).Should(BeEquivalentTo(4567))
			})

			It("includes the resource limits", func() {
				Expect(vcap_app).Should(HaveKey("limits"))
			})

			Describe("translation", func() {
				translations := map[string]string{
					"application_version": "version",
					"application_name":    "name",
					"application_uris":    "uris",
					"application_users":   "users",

					"started_at":           "start",
					"started_at_timestamp": "state_timestamp",
				}

				for k, v := range translations {
					It("should translate "+k+" to "+v, func() {

						Expect(vcap_app[k]).To(Equal(vcap_app[v]))
					})
				}
			})
		})

		Describe("exported_system_environment_variables", func() {
			JustBeforeEach(func() {
				exported_variables, _ = subject.ExportedSystemEnvironmentVariables()
			})

			it_exports("VCAP_APPLICATION", `"instance_index\\":37`)
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
			staging_message = testhelpers.Valid_staging_attributes()
			staging_message["start_message"] = start_message
		})

		JustBeforeEach(func() {
			baseDir, _ = ioutil.TempDir("", "env_test")
			config := &cfg.Config{BaseDir: baseDir}
			config.Staging.MaxStagingDuration = 900
			env := make(map[string]string)
			env["BUILDPACK_CACHE"] = ""
			config.Staging.Environment = env

			stgMsg := staging.NewStagingMessage(staging_message)
			staging_task := staging.NewStagingTask(config, stgMsg,
				[]dea.StagingBuildpack{}, nil, nil)
			subject = NewEnv(staging.NewStagingEnv(staging_task))
		})

		AfterEach(func() {
			os.RemoveAll(baseDir)
		})

		Describe("vcap_services", func() {
			var vcap_services map[string][]map[string]interface{}

			JustBeforeEach(func() {
				vcap_services = subject.VcapServices()
			})

			includesKey := func(key string) {
				It("includes "+key, func() {
					Expect(vcap_services[service["label"].(string)][0]).To(HaveKey(key))
				})
			}

			includesKey("name")
			includesKey("label")
			includesKey("tags")
			includesKey("plan")
			includesKey("plan_option")
			includesKey("credentials")

			It("doesn't include unknown keys", func() {
				label := service["label"].(string)
				Expect(vcap_services[label]).To(HaveLen(1))
				Expect(vcap_services[label][0]).ToNot(HaveKey("invalid"))
			})

			Describe("grouping", func() {
				BeforeEach(func() {
					services = []map[string]interface{}{
						merge(service, map[string]interface{}{"label": "l1"}),
						merge(service, map[string]interface{}{"label": "l1"}),
						merge(service, map[string]interface{}{"label": "l2"}),
					}
				})

				It("should group services by label", func() {
					Expect(vcap_services).To(HaveLen(2))
					Expect(vcap_services["l1"]).To(HaveLen(2))
					Expect(vcap_services["l2"]).To(HaveLen(1))
				})
			})

			Describe("ignoring", func() {
				BeforeEach(func() {
					services = []map[string]interface{}{
						merge(service, map[string]interface{}{"name": nil}),
					}
				})

				It("should ignore keys with nil values", func() {
					label := service["label"].(string)
					Expect(vcap_services[label]).To(HaveLen(1))
					Expect(vcap_services[label][0]).ToNot(HaveKey("name"))
				})
			})
		})

		Describe("vcap_application", func() {
			var vcap_app map[string]interface{}

			JustBeforeEach(func() {
				vcap_app = subject.VcapApplication()
			})

			includesKey := func(key string) {
				It("includes "+key, func() {
					Expect(vcap_app).To(HaveKey(key))
				})
			}

			includesKey("application_version")
			includesKey("application_name")
			includesKey("application_uris")

			It("includes the resource limits", func() {
				Expect(vcap_app).Should(HaveKey("limits"))
			})

			Describe("translation", func() {
				translations := map[string]string{
					"application_version": "version",
					"application_name":    "name",
					"application_uris":    "uris",
				}

				for k, v := range translations {
					It("should translate "+k+" to "+v, func() {
						Expect(vcap_app[k]).To(Equal(vcap_app[v]))
					})
				}
			})

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

func merge(s map[string]interface{}, m map[string]interface{}) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range s {
		r[k] = v
	}
	for k, v := range m {
		r[k] = v
	}

	return r
}
