package testhelpers

import (
	"dea/utils"
)

func valid_service_attributes(syslog_drain_url string) map[string]interface{} {
	return map[string]interface{}{
		"name":             "name",
		"type":             "type",
		"label":            "label",
		"vendor":           "vendor",
		"version":          "version",
		"tags":             []interface{}{"tag1", "tag2"},
		"plan":             "plan",
		"plan_option":      "plan_option",
		"syslog_drain_url": syslog_drain_url,
		"credentials": map[string]interface{}{
			"jdbcUrl":  "jdbc:mysql://some_user:some_password@some-db-provider.com:3306/db_name",
			"uri":      "mysql://some_user:some_password@some-db-provider.com:3306/db_name",
			"name":     "db_name",
			"hostname": "some-db-provider.com",
			"port":     "3306",
			"username": "some_user",
			"password": "some_password",
		},
	}
}

func Valid_instance_attributes(lots_of_services bool) map[string]interface{} {
	attrs := map[string]interface{}{
		"cc_partition": "partition",

		"instance_id":    utils.UUID(),
		"instance_index": 37,

		"application_id":      "37",
		"application_version": "some_version",
		"application_name":    "my_application",
		"application_uris":    []interface{}{"foo.com", "bar.com"},

		"droplet_sha1": "deadbeef",
		"droplet_uri":  "http://foo.com/file.ext",

		"limits": map[string]interface{}{"mem": 512, "disk": 128, "fds": 5000},
		"env":    []interface{}{"FOO=BAR"},

		"warden_handle":           "1234",
		"instance_host_port":      float64(2345),
		"instance_container_port": float64(4567),
	}

	services := make([]map[string]interface{}, 0, 3)
	attrs["services"] = services
	if lots_of_services {
		services = append(services, valid_service_attributes("syslog://log.example.com"))
		services = append(services, valid_service_attributes(""))
		services = append(services, valid_service_attributes("syslog://log2.example.com"))
	} else {
		services = append(services, valid_service_attributes(""))
	}

	return attrs
}

func Valid_staging_attributes() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"services":    []interface{}{},
			"environment": []interface{}{"FOO=BAR"},
			"resources": map[string]interface{}{
				"memory": 512,
				"disk":   128,
				"fds":    5000,
			},
		},
		"app_id":           "app-guid",
		"task_id":          utils.UUID(),
		"download_uri":     "http://127.0.0.1:12346/download",
		"upload_uri":       "http://127.0.0.1:12346/upload",
		"staged_path":      "",
		"start_message":    Valid_instance_attributes(false),
		"admin_buildpacks": []map[string]interface{}{},
	}
}
