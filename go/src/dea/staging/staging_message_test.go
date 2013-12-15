package staging

import (
	"dea"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/url"
)

var _ = Describe("StagingMessage", func() {
	var admin_buildpacks []map[string]interface{}
	var stagingMsg *StagingMessage

	start_message := map[string]interface{}{
		"droplet":        "droplet-id",
		"name":           "name",
		"uris":           []interface{}{"tnky1j0-buildpack-test.a1-app.cf-app.com"},
		"prod":           false,
		"sha1":           nil,
		"executableFile": "deprecated",
		"executableUri":  nil,
		"version":        "version-number",
		"services":       []map[string]interface{}{},
		"limits": map[string]interface{}{
			"mem":  64,
			"disk": 1024,
			"fds":  16384,
		},
		"cc_partition":  "default",
		"env":           []interface{}{},
		"console":       false,
		"debug":         nil,
		"start_command": nil,
		"index":         0,
	}

	staging_message := map[string]interface{}{
		"app_id":                       "some-node-app-id",
		"task_id":                      "task-id",
		"properties":                   map[string]interface{}{"some_property": "some_value"},
		"download_uri":                 "http://localhost/unstaged/rails3_with_db",
		"upload_uri":                   "http://localhost/upload/rails3_with_db",
		"buildpack_cache_download_uri": "http://localhost/buildpack_cache/download",
		"buildpack_cache_upload_uri":   "http://localhost/buildpack_cache/upload",
		"admin_buildpacks":             admin_buildpacks,
		"start_message":                start_message,
	}

	BeforeEach(func() {
		admin_buildpacks = []map[string]interface{}{}
	})

	JustBeforeEach(func() {
		staging_message["admin_buildpacks"] = admin_buildpacks
		stagingMsg = NewStagingMessage(staging_message)
	})

	It("has correct values", func() {
		Expect(stagingMsg.App_id()).To(Equal("some-node-app-id"))
		Expect(stagingMsg.Task_id()).To(Equal("task-id"))
		Expect(stagingMsg.Download_uri().String()).To(Equal("http://localhost/unstaged/rails3_with_db"))
		Expect(stagingMsg.Upload_uri().String()).To(Equal("http://localhost/upload/rails3_with_db"))
		Expect(stagingMsg.Buildpack_cache_upload_uri().String()).To(Equal("http://localhost/buildpack_cache/upload"))
		Expect(stagingMsg.Buildpack_cache_download_uri().String()).To(Equal("http://localhost/buildpack_cache/download"))
		Expect(stagingMsg.AdminBuildpacks()).To(Equal([]dea.StagingBuildpack{}))
		Expect(stagingMsg.Properties()).To(Equal(map[string]interface{}{"some_property": "some_value"}))
		Expect(stagingMsg.AsMap()).To(Equal(staging_message))
	})

	Context("when admin build packs are specified", func() {
		BeforeEach(func() {
			admin_buildpacks = []map[string]interface{}{
				{"url": "http://www.example.com/buildpacks/uri/first", "key": "first"},
				{"url": "http://www.example.com/buildpacks/uri/second", "key": "second"},
			}
		})

		It("has admin buildpacks", func() {
			url1, _ := url.Parse(admin_buildpacks[0]["url"].(string))
			url2, _ := url.Parse(admin_buildpacks[1]["url"].(string))
			Expect(stagingMsg.AdminBuildpacks()).To(Equal([]dea.StagingBuildpack{
				{url1, "first"},
				{url2, "second"},
			}))
		})
	})
})
