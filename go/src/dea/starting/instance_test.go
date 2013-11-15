package starting_test

import (
	"dea/config"
	"dea/starting"
	"dea/testhelpers"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultAttributes(t *testing.T) {
	instance := newInstance(nil)
	assert.Equal(t, instance.ExitStatus(), -1)
}

func TestInstanceAttributes(t *testing.T) {
	instance := newInstance(map[string]interface{}{
		"index": 37,
	})
	assert.NotNil(t, instance.Id())
	assert.Equal(t, instance.Index(), 37)
}

func TestApplicationAttributes(t *testing.T) {
	instance := newInstance(map[string]interface{}{
		"droplet": 37,
		"version": "some_version",
		"name":    "my_application",
		"uris":    []string{"foo.com", "bar.com"},
		"users":   []string{"john@doe.com"},
	})
	assert.Equal(t, instance.ApplicationId(), "37")
	assert.Equal(t, instance.ApplicationVersion(), "some_version")
	assert.Equal(t, instance.ApplicationName(), "my_application")
	assert.Equal(t, instance.ApplicationUris(), []string{"foo.com", "bar.com"})
}

func TestDropletAttributes(t *testing.T) {
	instance := newInstance(map[string]interface{}{
		"sha1":          "deadbeef",
		"executableUri": "http://foo.com/file.ext",
	})
	assert.Equal(t, instance.DropletSHA1(), "deadbeef")
	assert.Equal(t, instance.DropletUri(), "http://foo.com/file.ext")
}

func TestStartCommand(t *testing.T) {
	instance := newInstance(map[string]interface{}{
		"start_command": "start command",
	})
	assert.Equal(t, instance.StartCommand(), "start command")
}

func TestNilStartCommand(t *testing.T) {
	instance := newInstance(map[string]interface{}{
		"start_command": "",
	})
	assert.Equal(t, instance.StartCommand(), "")
}

func TestMissingStartCommand(t *testing.T) {
	instance := newInstance(map[string]interface{}{})
	assert.Equal(t, instance.StartCommand(), "")
}

func TestOtherAttributes(t *testing.T) {
	limits := starting.LimitsData{MemMb: 1, DiskMb: 2, Fds: 3}
	limitsMap := make(map[string]uint64)
	limits.MarshalMap(limitsMap)

	service := starting.ServiceData{Name: "redis", Label: "redis"}
	serviceMap := make(map[string]interface{})
	service.MarshalMap(serviceMap)
	services := []map[string]interface{}{serviceMap}
	instance := newInstance(map[string]interface{}{
		"limits":   limitsMap,
		"env":      []interface{}{"FOO=BAR", "BAR=", "QUX"},
		"services": services,
	})

	assert.Equal(t, instance.Limits(), limits)
	assert.Equal(t, instance.Environment(), map[string]string{"FOO": "BAR", "BAR": "", "QUX": ""})
	assert.Equal(t, instance.Services(), []starting.ServiceData{service})
}

func TestMemoryLimit(t *testing.T) {
	instance := newInstance(testhelpers.Valid_instance_attributes(false))
	assert.Equal(t, instance.MemoryLimit(), 512*config.Mebi)
}

func TestDiskLimit(t *testing.T) {
	instance := newInstance(testhelpers.Valid_instance_attributes(false))
	assert.Equal(t, instance.DiskLimit(), 128*config.MB)
}

func TestFileDescriptorLimit(t *testing.T) {
	instance := newInstance(testhelpers.Valid_instance_attributes(false))
	assert.Equal(t, instance.FileDescriptorLimit(), uint64(5000))
}

func newInstance(merge map[string]interface{}) *starting.Instance {
	attrs := testhelpers.Valid_instance_attributes(false)
	if merge != nil {
		for k, v := range merge {
			attrs[k] = v
		}
	}

	return starting.NewInstance(attrs, &config.Config{}, nil, "127.0.0.1")
}
