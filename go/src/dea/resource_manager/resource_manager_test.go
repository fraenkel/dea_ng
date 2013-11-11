package resource_manager

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultConfigResourceManager(t *testing.T) {
	resourceManager := NewResourceManager(nil, nil, &ResourcesConfig{})

	assert.Equal(t, resourceManager.MemoryCapacity(), 8*1024*1.0)
	assert.Equal(t, resourceManager.DiskCapacity(), 16*1024*1024*1.0)
}

func TestConfiguredResourceManager(t *testing.T) {
	resourceManager := NewResourceManager(nil, nil,
		&ResourcesConfig{
			MemoryMb:               4 * 1024,
			MemoryOvercommitFactor: 1.5,
			DiskMb:                 8 * 1024 * 1024,
			DiskOvercommitFactor:   1.5,
		})

	assert.Equal(t, resourceManager.MemoryCapacity(), 4*1024*1.5)
	assert.Equal(t, resourceManager.DiskCapacity(), 8*1024*1024*1.5)
}

func TestPartialConfigResourceManager(t *testing.T) {
	resourceManager := NewResourceManager(nil, nil,
		&ResourcesConfig{
			MemoryMb:               4 * 1024,
			DiskOvercommitFactor:   1.5,
		})
	
	assert.Equal(t, resourceManager.MemoryCapacity(), 4*1024*1.0)
	assert.Equal(t, resourceManager.DiskCapacity(), 16*1024*1024*1.5)
}
