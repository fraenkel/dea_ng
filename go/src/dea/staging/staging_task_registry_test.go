package staging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var config = map[string]interface{}{}
var task1 = NewStagingTask(config, valid_staging_attributes(), []string{})
var task2 = NewStagingTask(config, valid_staging_attributes(), []string{})
var buildpacks = []string{}

func TestRegister(t *testing.T) {
	stagingRegistry := NewStagingTaskRegistry()
	stagingRegistry.Register(&task1)
	assert.Equal(t, len(stagingRegistry.tasks), 1)
}

func TestUnregisterWhenPreviouslyRegistered(t *testing.T) {
	stagingRegistry := NewStagingTaskRegistry()
	stagingRegistry.Register(&task1)

	stagingRegistry.Unregister(&task1)
	assert.Equal(t, len(stagingRegistry.tasks), 0)
}

func TestUnregisterWhenNotRegistered(t *testing.T) {
	stagingRegistry := NewStagingTaskRegistry()

	stagingRegistry.Unregister(&task1)
	assert.Equal(t, len(stagingRegistry.tasks), 0)
}

func TestTaskRetrieval(t *testing.T) {
	stagingRegistry := NewStagingTaskRegistry()
	stagingRegistry.Register(&task1)

	assert.Equal(t, stagingRegistry.Task(task1.id), &task1)
}

func TestTasks(t *testing.T) {
	stagingRegistry := NewStagingTaskRegistry()
	stagingRegistry.Register(&task1)
	stagingRegistry.Register(&task2)

	tasks := stagingRegistry.Tasks()
	assert.Equal(t, 2, len(tasks))
}
