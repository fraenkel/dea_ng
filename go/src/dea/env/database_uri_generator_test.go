package env_test

import (
	. "dea/env"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DatabaseUriGenerator.Go", func() {
	var services_env []map[string]interface{}
	var services *DatabaseUriGenerator
	var expectedUri string

	Context("map[string]string", func() {
		var db_env map[string]string

		BeforeEach(func() {
			db_env = map[string]string{"uri": "postgres://username:password@host/db"}
			services_env = []map[string]interface{}{
				map[string]interface{}{"credentials": db_env}}

		})

		JustBeforeEach(func() {
			services = NewDatabaseUriGenerator(services_env)
		})

		AssertDBUriBehavior := func() {
			It("db uri", func() {
				databaseUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(databaseUri.String()).To(Equal(expectedUri))
			})
		}

		Context("mysql", func() {
			BeforeEach(func() {
				db_env["uri"] = "mysql://username:password@host/db"
				expectedUri = "mysql2://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("mysq2", func() {
			BeforeEach(func() {
				db_env["uri"] = "mysql2://username:password@host/db"
				expectedUri = "mysql2://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("postgres", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgres://username:password@host/db"
				expectedUri = "postgres://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("postgresql", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgresql://username:password@host/db"
				expectedUri = "postgres://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("and there are more than one production relational database", func() {
			BeforeEach(func() {
				services_env = []map[string]interface{}{
					map[string]interface{}{"name": "first_db", "credentials": map[string]string{"uri": "postgres://username:password@host/db1"}},
					map[string]interface{}{"name": "second_db", "credentials": map[string]string{"uri": "postgres://username:password@host/db2"}}}

				expectedUri = "postgres://username:password@host/db1"
			})

			AssertDBUriBehavior()
		})

		Context("and the uri is invalid", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgresql:///inva%!:password@host/db"
				expectedUri = ""
			})

			It("raises an error", func() {
				_, err := services.DatabaseUri()
				Expect(err).NotTo(BeNil())
			})
		})

		Context("when there are non relational databse services", func() {
			BeforeEach(func() {
				db_env["uri"] = "sendgrid://foo:bar@host/db"
				expectedUri = ""
			})

			It("returns nil", func() {
				dbUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(dbUri).To(BeNil())
			})

		})

		Context("when there are no services", func() {
			BeforeEach(func() {
				services_env = nil
				expectedUri = ""
			})
			It("returns nil", func() {
				dbUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(dbUri).To(BeNil())
			})
		})
	})

	Context("map[string]interface{}", func() {
		var db_env map[string]interface{}

		BeforeEach(func() {
			db_env = map[string]interface{}{"uri": "postgres://username:password@host/db"}
			services_env = []map[string]interface{}{
				map[string]interface{}{"credentials": db_env}}

		})

		JustBeforeEach(func() {
			services = NewDatabaseUriGenerator(services_env)
		})

		AssertDBUriBehavior := func() {
			It("db uri", func() {
				databaseUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(databaseUri.String()).To(Equal(expectedUri))
			})
		}

		Context("mysql", func() {
			BeforeEach(func() {
				db_env["uri"] = "mysql://username:password@host/db"
				expectedUri = "mysql2://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("mysq2", func() {
			BeforeEach(func() {
				db_env["uri"] = "mysql2://username:password@host/db"
				expectedUri = "mysql2://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("postgres", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgres://username:password@host/db"
				expectedUri = "postgres://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("postgresql", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgresql://username:password@host/db"
				expectedUri = "postgres://username:password@host/db"
			})

			AssertDBUriBehavior()
		})

		Context("and there are more than one production relational database", func() {
			BeforeEach(func() {
				services_env = []map[string]interface{}{
					map[string]interface{}{"name": "production", "credentials": map[string]string{"uri": "postgres://username:password@host/db1"}},
					map[string]interface{}{"name": "prod", "credentials": map[string]string{"uri": "postgres://username:password@host/db2"}}}

				expectedUri = "postgres://username:password@host/db1"
			})

			AssertDBUriBehavior()
		})

		Context("and the uri is invalid", func() {
			BeforeEach(func() {
				db_env["uri"] = "postgresql:///inva%!:password@host/db"
				expectedUri = ""
			})

			It("raises an error", func() {
				_, err := services.DatabaseUri()
				Expect(err).NotTo(BeNil())
			})
		})

		Context("when there are non relational databse services", func() {
			BeforeEach(func() {
				db_env["uri"] = "sendgrid://foo:bar@host/db"
				expectedUri = ""
			})

			It("returns nil", func() {
				dbUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(dbUri).To(BeNil())
			})

		})

		Context("when there are no services", func() {
			BeforeEach(func() {
				services_env = nil
				expectedUri = ""
			})
			It("returns nil", func() {
				dbUri, err := services.DatabaseUri()
				Expect(err).To(BeNil())
				Expect(dbUri).To(BeNil())
			})
		})
	})
})
